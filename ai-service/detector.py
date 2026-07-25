"""
detector.py
===========
Punto di ingresso del modulo Computer Vision / AI.

Pipeline:
  webcam --> OpenCV (acquisizione frame)
         --> YOLOv8 (object detection, classe "person")
         --> CentroidTracker (assegna un ID stabile tra i frame, nei
             limiti di un tracker basato su centroide: nessun
             riconoscimento facciale, solo continuita' finche' la
             persona resta nel campo visivo)
         --> event_client (invio evento al backend Go via HTTP)
         --> streaming del frame annotato verso la dashboard

Il backend e' l'unica fonte di verita' su nome e stato "critica" di
ogni persona: il modulo AI mantiene solo una cache locale (rinfrescata
periodicamente) per colorare il riquadro nel feed, ma la classificazione
NORMAL/CRITICAL dell'evento viene decisa sul server.

Esegui con:  python detector.py
"""

import base64
import logging
import threading
import time
import uuid
from datetime import datetime, timezone

import cv2
import requests
from ultralytics import YOLO

import config
from tracker import CentroidTracker
from event_client import send_event

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("detector")

KNOWN_PERSONS = {}          # track_id (str) -> {"nome":..., "is_critical":...}
_known_persons_lock = threading.Lock()
last_appearance = {}


def load_known_persons():
    """Ricarica la cache locale di persone note dal backend.
    Usa l'endpoint /api/internal/persons, protetto dal token di servizio
    (il modulo AI non fa login utente, quindi non ha un JWT)."""
    try:
        resp = requests.get(
            f"{config.BACKEND_URL}/api/internal/persons",
            headers={"Authorization": f"Bearer {config.API_TOKEN}"},
            timeout=3.0,
        )
        if resp.ok:
            fresh = {p["track_id"]: p for p in resp.json()}
            with _known_persons_lock:
                KNOWN_PERSONS.clear()
                KNOWN_PERSONS.update(fresh)
            logger.info(f"Cache persone aggiornata: {len(fresh)} persone note")
        else:
            logger.warning(f"Backend ha rifiutato la richiesta persone ({resp.status_code}): {resp.text}")
    except requests.RequestException as e:
        logger.warning(f"Impossibile ricaricare le persone note dal backend: {e}")


def known_persons_refresh_loop():
    """Thread di sfondo: tiene la cache locale allineata al backend, cosi'
    una rinomina o un cambio di stato "critica" fatti dalla dashboard si
    riflettono nel feed video senza dover riavviare il modulo AI."""
    while True:
        time.sleep(config.KNOWN_PERSONS_REFRESH_SECONDS)
        load_known_persons()


def next_tracker_start_id():
    """Calcola da quale ID far ripartire il tracker, in base al track_id
    numerico piu' alto gia' noto al backend (evita di riassegnare subito
    un ID gia' presente in database a una persona diversa dopo un riavvio)."""
    max_id = -1
    with _known_persons_lock:
        for track_id in KNOWN_PERSONS.keys():
            try:
                max_id = max(max_id, int(track_id))
            except ValueError:
                continue
    return max_id + 1


def should_send_new_appearance(track_id, last_sent_at):
    now = time.time()
    if track_id not in last_appearance or (now - last_appearance.get(track_id, 0) > config.NEW_APPEARANCE_GAP_SECONDS):
        last_sent_at[track_id] = now
        last_appearance[track_id] = now
        return True
    last_appearance[track_id] = now
    return False


def stream_frame(frame):
    """Codifica il frame in JPEG e lo invia al backend, che lo ridistribuisce
    alla dashboard via WebSocket per il feed live."""
    ok, buf = cv2.imencode(".jpg", frame, [cv2.IMWRITE_JPEG_QUALITY, config.FRAME_JPEG_QUALITY])
    if not ok:
        return
    b64 = base64.b64encode(buf).decode("ascii")
    try:
        requests.post(
            config.FRAME_ENDPOINT,
            json={"camera": config.CAMERA_CODE, "data": b64},
            headers={"Authorization": f"Bearer {config.API_TOKEN}"},
            timeout=1.5,
        )
    except requests.RequestException:
        pass  # il feed non deve mai bloccare il loop di detection


def main():
    logger.info("=== AI MONITORING - SINGOLA TELECAMERA ===")
    load_known_persons()

    threading.Thread(target=known_persons_refresh_loop, daemon=True).start()

    model = YOLO(config.MODEL_WEIGHTS)
    tracker = CentroidTracker(max_disappeared=40, max_distance=80, start_id=next_tracker_start_id())

    cap = cv2.VideoCapture(int(config.CAMERA_SOURCE) if str(config.CAMERA_SOURCE).isdigit() else config.CAMERA_SOURCE)
    if not cap.isOpened():
        logger.error("Impossibile aprire la telecamera")
        return

    frame_count = 0
    last_sent_at = {}
    last_frame_streamed_at = 0.0

    try:
        while True:
            ok, frame = cap.read()
            if not ok:
                time.sleep(0.5)
                continue

            frame_count += 1
            if frame_count % config.PROCESS_EVERY_N_FRAMES != 0:
                continue

            results = model(frame, verbose=False)[0]

            rects = []
            confidences = []
            for box in results.boxes:
                if int(box.cls[0]) != config.PERSON_CLASS_ID or float(box.conf[0]) < config.CONFIDENCE_THRESHOLD:
                    continue
                x1, y1, x2, y2 = map(int, box.xyxy[0])
                rects.append((x1, y1, x2, y2))
                confidences.append(float(box.conf[0]))

            tracked = tracker.update(rects)
            now = datetime.now(timezone.utc)

            for track_id, (centroid, bbox) in tracked.items():
                with _known_persons_lock:
                    person_info = KNOWN_PERSONS.get(str(track_id), {})
                is_critical = bool(person_info.get("is_critical"))
                nome = person_info.get("nome") or f"Persona {track_id}"
                color = (0, 0, 255) if is_critical else (0, 255, 0)

                x1, y1, x2, y2 = bbox
                cv2.rectangle(frame, (x1, y1), (x2, y2), color, 3)
                label = f"{nome}" + (" - CRITICO" if is_critical else "")
                cv2.putText(frame, label, (x1, max(y1 - 10, 15)), cv2.FONT_HERSHEY_SIMPLEX, 0.6, color, 2)

                if should_send_new_appearance(track_id, last_sent_at):
                    confidence = confidences[0] if confidences else 0.85
                    payload = {
                        "event_id": str(uuid.uuid4()),
                        "camera": config.CAMERA_CODE,
                        "track_id": str(track_id),
                        "confidence": confidence,
                        "bbox": {"x1": bbox[0], "y1": bbox[1], "x2": bbox[2], "y2": bbox[3]},
                        "timestamp": now.isoformat(),
                    }
                    send_event(payload)

            # invia il frame annotato alla dashboard, con throttling indipendente
            # dal frame processing (non deve rallentare la detection)
            t = time.time()
            if t - last_frame_streamed_at >= config.FRAME_STREAM_MIN_INTERVAL:
                stream_frame(frame)
                last_frame_streamed_at = t

            if config.SHOW_PREVIEW_WINDOW:
                cv2.imshow("AI Monitoring", frame)
                if cv2.waitKey(1) & 0xFF == ord("q"):
                    break

    except KeyboardInterrupt:
        logger.info("Chiusura")
    finally:
        cap.release()
        cv2.destroyAllWindows()


if __name__ == "__main__":
    main()
