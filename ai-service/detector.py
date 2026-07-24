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

KNOWN_PERSONS = {}  # track_id -> info

def load_known_persons():
    global KNOWN_PERSONS
    try:
        resp = requests.get(
            f"{config.BACKEND_URL}/api/persons",
            headers={"Authorization": f"Bearer {config.API_TOKEN}"}
        )
        if resp.ok:
            for p in resp.json():
                KNOWN_PERSONS[p["track_id"]] = p
    except:
        logger.warning("Impossibile caricare known persons")

def update_person_critical(track_id, is_critical):
    try:
        requests.patch(
            f"{config.BACKEND_URL}/api/persons/{track_id}",
            json={"is_critical": is_critical},
            headers={"Authorization": f"Bearer {config.API_TOKEN}"}
        )
    except:
        pass

def should_send_new_appearance(track_id, last_sent_at):
    now = time.time()
    last = last_sent_at.get(track_id, 0)
    if now - last < config.SEND_MIN_INTERVAL_SECONDS:
        return False
    last_sent_at[track_id] = now
    return True

def main():
    logger.info("Avvio AI Monitoring - Versione Finale")
    load_known_persons()

    model = YOLO(config.MODEL_WEIGHTS)
    tracker = CentroidTracker(max_disappeared=40, max_distance=80)
    anomaly_detector = AnomalyDetector(... )  # mantieni come prima

    cap = cv2.VideoCapture(int(config.CAMERA_SOURCE) if str(config.CAMERA_SOURCE).isdigit() else config.CAMERA_SOURCE)
    
    frame_count = 0
    last_sent_at = {}

    try:
        while True:
            # ... (logica di lettura frame e detection invariata) ...

            for track_id, (centroid, bbox) in tracked.items():
                person_info = KNOWN_PERSONS.get(str(track_id), {"is_critical": False, "nome": f"Person_{track_id}"})
                color = (0, 0, 255) if person_info.get("is_critical") else (0, 255, 0)

                if config.SHOW_PREVIEW_WINDOW:
                    x1,y1,x2,y2 = bbox
                    cv2.rectangle(frame, (x1,y1),(x2,y2), color, 3)
                    cv2.putText(frame, f"ID {track_id} {'CRITICO' if person_info.get('is_critical') else ''}", 
                                (x1, y1-10), cv2.FONT_HERSHEY_SIMPLEX, 0.6, color, 2)

                if should_send_new_appearance(track_id, last_sent_at):
                    severity = "critical" if person_info.get("is_critical") else "info"
                    payload = { ... }  # come prima
                    send_event(payload)

            if config.SHOW_PREVIEW_WINDOW:
                cv2.imshow("AI Monitoring", frame)
                if cv2.waitKey(1) & 0xFF == ord("q"):
                    break

    except KeyboardInterrupt:
        pass
    finally:
        cap.release()
        cv2.destroyAllWindows()

if __name__ == "__main__":
    main()
