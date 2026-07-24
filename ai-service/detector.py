"""
detector.py
===========
Punto di ingresso del modulo Computer Vision / AI.

Pipeline:
  webcam --> OpenCV (acquisizione frame)
         --> YOLOv8 (object detection, classe "person")
         --> CentroidTracker (assegna un ID stabile tra i frame)
         --> Zone / AnomalyDetector (regole di business)
         --> event_client (invio evento al backend Go via HTTP)

Esegui con:  python detector.py
"""

import logging
import time
import uuid
import requests
from datetime import datetime, timezone

import cv2
from ultralytics import YOLO

import config
from tracker import CentroidTracker
from zones import AnomalyDetector
from event_client import send_event

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("detector")

KNOWN_PERSONS = {}
last_appearance = {}

def load_known_persons():
    global KNOWN_PERSONS
    try:
        resp = requests.get(f"{config.BACKEND_URL}/api/persons")
        if resp.ok:
            for p in resp.json():
                KNOWN_PERSONS[p["track_id"]] = p
        logger.info(f"Caricate {len(KNOWN_PERSONS)} persone note")
    except:
        logger.warning("Impossibile caricare persone dal backend")

def should_send_new_appearance(track_id, last_sent_at):
    global last_appearance
    now = time.time()
    
    # Nuova apparizione se era scomparsa da > 3 secondi
    if track_id not in last_appearance or (now - last_appearance.get(track_id, 0) > 3.0):
        last_sent_at[track_id] = now
        last_appearance[track_id] = now
        return True
    
    last_appearance[track_id] = now
    return False

def main():
    logger.info("=== AI MONITORING - SINGOLA TELECAMERA ===")
    load_known_persons()

    model = YOLO(config.MODEL_WEIGHTS)
    tracker = CentroidTracker(max_disappeared=40, max_distance=80)

    from datetime import time as dtime
    start_h, start_m = map(int, config.ALLOWED_HOUR_START.split(":"))
    end_h, end_m = map(int, config.ALLOWED_HOUR_END.split(":"))
    anomaly_detector = AnomalyDetector(
        allowed_start=dtime(start_h, start_m),
        allowed_end=dtime(end_h, end_m)
    )

    cap = cv2.VideoCapture(int(config.CAMERA_SOURCE) if str(config.CAMERA_SOURCE).isdigit() else config.CAMERA_SOURCE)
    if not cap.isOpened():
        logger.error("Impossibile aprire la telecamera")
        return

    frame_count = 0
    last_sent_at = {}

    try:
        while True:
            ok, frame = cap.read()
            if not ok:
                time.sleep(0.5)
                continue

            frame_count += 1
            if frame_count % config.PROCESS_EVERY_N_FRAMES != 0:
                continue

            h, w = frame.shape[:2]
            results = model(frame, verbose=False)[0]

            rects = []
            for box in results.boxes:
                if int(box.cls[0]) != config.PERSON_CLASS_ID or float(box.conf[0]) < config.CONFIDENCE_THRESHOLD:
                    continue
                x1, y1, x2, y2 = map(int, box.xyxy[0])
                rects.append((x1, y1, x2, y2))

            tracked = tracker.update(rects)

            now = datetime.now(timezone.utc)

            for track_id, (centroid, bbox) in tracked.items():
                person_info = KNOWN_PERSONS.get(str(track_id), {"is_critical": False})
                color = (0, 0, 255) if person_info.get("is_critical") else (0, 255, 0)

                if config.SHOW_PREVIEW_WINDOW:
                    x1, y1, x2, y2 = bbox
                    cv2.rectangle(frame, (x1, y1), (x2, y2), color, 3)
                    label = f"ID {track_id} {'CRITICO' if person_info.get('is_critical') else ''}"
                    cv2.putText(frame, label, (x1, y1-10), cv2.FONT_HERSHEY_SIMPLEX, 0.6, color, 2)

                if should_send_new_appearance(track_id, last_sent_at):
                    severity = "critical" if person_info.get("is_critical") else "info"
                    payload = {
                        "event_id": str(uuid.uuid4()),
                        "camera": config.CAMERA_CODE,
                        "tipo_evento": "person_detected",
                        "severity": severity,
                        "track_id": str(track_id),
                        "confidence": 0.85,
                        "bbox": {"x1": bbox[0], "y1": bbox[1], "x2": bbox[2], "y2": bbox[3]},
                        "timestamp": now.isoformat(),
                        "metadata": {"is_critical": person_info.get("is_critical")}
                    }
                    send_event(payload)

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
