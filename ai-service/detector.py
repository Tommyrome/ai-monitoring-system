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
from datetime import datetime

import cv2
from ultralytics import YOLO

import config
from tracker import CentroidTracker
from zones import Zone, AnomalyDetector
from event_client import send_event

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("detector")


def build_zones():
    return [Zone(z["name"], z["polygon"], z["type"]) for z in config.ZONES_CONFIG]


def classify_event(zone_alert: bool, anomaly: bool):
    """Restituisce (tipo_evento, severity) secondo lo schema INFO/WARNING/CRITICAL."""
    if zone_alert:
        return "zone_alert", "critical"
    if anomaly:
        return "anomaly", "warning"
    return "person_detected", "info"


def main():
    logger.info("Avvio modulo AI - camera=%s modello=%s", config.CAMERA_CODE, config.MODEL_WEIGHTS)

    model = YOLO(config.MODEL_WEIGHTS)
    tracker = CentroidTracker(max_disappeared=30, max_distance=80)
    zones = build_zones()

    start_h, start_m = map(int, config.ALLOWED_HOUR_START.split(":"))
    end_h, end_m = map(int, config.ALLOWED_HOUR_END.split(":"))
    from datetime import time as dtime
    anomaly_detector = AnomalyDetector(
        allowed_start=dtime(start_h, start_m),
        allowed_end=dtime(end_h, end_m),
        max_people_per_zone=config.MAX_PEOPLE_PER_ZONE,
        max_dwell_seconds=config.MAX_DWELL_SECONDS,
    )

    source = int(config.CAMERA_SOURCE) if config.CAMERA_SOURCE.isdigit() else config.CAMERA_SOURCE
    cap = cv2.VideoCapture(source)
    if not cap.isOpened():
        logger.error("Impossibile aprire la sorgente video: %s", config.CAMERA_SOURCE)
        return

    frame_count = 0
    last_sent_at = {}  # track_id -> timestamp ultimo invio (throttling)

    try:
        while True:
            ok, frame = cap.read()
            if not ok:
                logger.warning("Frame non disponibile, riprovo...")
                time.sleep(0.5)
                continue

            frame_count += 1
            if frame_count % config.PROCESS_EVERY_N_FRAMES != 0:
                continue

            h, w = frame.shape[:2]
            results = model(frame, verbose=False)[0]

            rects = []
            confidences = {}
            for box in results.boxes:
                cls_id = int(box.cls[0])
                conf = float(box.conf[0])
                if cls_id != config.PERSON_CLASS_ID or conf < config.CONFIDENCE_THRESHOLD:
                    continue
                x1, y1, x2, y2 = map(int, box.xyxy[0])
                rects.append((x1, y1, x2, y2))
                confidences[(x1, y1, x2, y2)] = conf

            tracked = tracker.update(rects)

            # conteggio persone per zona (per l'overcrowding check)
            people_per_zone = {z.name: 0 for z in zones}

            now = datetime.now()
            out_of_hours = anomaly_detector.check_out_of_hours(now)

            for track_id, (centroid, bbox) in tracked.items():
                conf = confidences.get(bbox, 0.0)

                zone_alert = False
                anomaly = False
                zone_hit = None

                for zone in zones:
                    if zone.contains(centroid, w, h):
                        zone_hit = zone
                        people_per_zone[zone.name] += 1
                        if zone.zone_type == "restricted":
                            zone_alert = True
                        if anomaly_detector.check_dwell_time(zone.name, track_id, now):
                            anomaly = True

                if out_of_hours:
                    anomaly = True

                # throttling: non inviare lo stesso track_id troppo frequentemente
                last = last_sent_at.get(track_id, 0)
                if time.time() - last < config.SEND_MIN_INTERVAL_SECONDS and not zone_alert:
                    continue
                last_sent_at[track_id] = time.time()

                event_type, severity = classify_event(zone_alert, anomaly)

                payload = {
                    "event_id": str(uuid.uuid4()),
                    "camera": config.CAMERA_CODE,
                    "tipo_evento": event_type,
                    "severity": severity,
                    "track_id": str(track_id),
                    "confidence": round(conf, 3),
                    "bbox": {"x1": bbox[0], "y1": bbox[1], "x2": bbox[2], "y2": bbox[3]},
                    "zone": zone_hit.name if zone_hit else None,
                    "timestamp": now.isoformat(),
                }
                sent = send_event(payload)
                logger.info(
                    "Evento %s track=%s conf=%.2f severity=%s inviato=%s",
                    event_type, track_id, conf, severity, sent,
                )

            # overcrowding check dopo aver contato tutte le persone per zona
            for zone in zones:
                if anomaly_detector.check_overcrowding(people_per_zone[zone.name]):
                    payload = {
                        "event_id": str(uuid.uuid4()),
                        "camera": config.CAMERA_CODE,
                        "tipo_evento": "anomaly",
                        "severity": "warning",
                        "track_id": None,
                        "confidence": 1.0,
                        "bbox": None,
                        "zone": zone.name,
                        "timestamp": now.astimezone().isoformat(),
                        "metadata": {"reason": "overcrowding", "count": people_per_zone[zone.name]},
                    }
                    send_event(payload)
                    logger.info("Anomalia overcrowding in zona %s (%d persone)", zone.name, people_per_zone[zone.name])

            if config.SHOW_PREVIEW_WINDOW:
                for track_id, (centroid, bbox) in tracked.items():
                    x1, y1, x2, y2 = bbox
                    cv2.rectangle(frame, (x1, y1), (x2, y2), (0, 255, 0), 2)
                    cv2.putText(frame, f"ID {track_id}", (x1, y1 - 8),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 2)
                for zone in zones:
                    poly = zone.polygon_pixels(w, h)
                    cv2.polylines(frame, [poly], True, (0, 0, 255), 2)
                cv2.imshow("AI Camera Monitoring - preview", frame)
                if cv2.waitKey(1) & 0xFF == ord("q"):
                    break

    except KeyboardInterrupt:
        logger.info("Interruzione richiesta dall'utente")
    finally:
        cap.release()
        cv2.destroyAllWindows()


if __name__ == "__main__":
    main()
