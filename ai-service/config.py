import os
from dotenv import load_dotenv

load_dotenv()

# --- Backend ---
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")
EVENTS_ENDPOINT = f"{BACKEND_URL}/api/events"
API_TOKEN = os.getenv("AI_SERVICE_TOKEN", "change-me-shared-secret")

# --- Camera ---
CAMERA_SOURCE = os.getenv("CAMERA_SOURCE", "0")  # 0 = webcam, oppure path/RTSP url
CAMERA_CODE = os.getenv("CAMERA_CODE", "camera_01")

# --- Modello YOLO ---
MODEL_WEIGHTS = os.getenv("MODEL_WEIGHTS", "yolov8n.pt")  # nano: veloce, ideale per demo/CPU
CONFIDENCE_THRESHOLD = float(os.getenv("CONFIDENCE_THRESHOLD", "0.5"))
PERSON_CLASS_ID = 0  # nel dataset COCO, "person" ha class id 0

# --- Frame processing ---
PROCESS_EVERY_N_FRAMES = int(os.getenv("PROCESS_EVERY_N_FRAMES", "2"))  # per alleggerire la CPU
SEND_MIN_INTERVAL_SECONDS = float(os.getenv("SEND_MIN_INTERVAL_SECONDS", "1.0"))  # throttling eventi

# --- Zone di esempio (poligoni normalizzati 0..1) ---
# In produzione queste andrebbero caricate dinamicamente dal backend (GET /api/cameras/{id}/zones)
ZONES_CONFIG = [
    {
        "name": "zona_vietata",
        "type": "restricted",
        "polygon": [(0.6, 0.1), (1.0, 0.1), (1.0, 0.6), (0.6, 0.6)],
    },
]

# --- Anomaly detection ---
ALLOWED_HOUR_START = "06:00"
ALLOWED_HOUR_END = "22:00"
MAX_PEOPLE_PER_ZONE = 5
MAX_DWELL_SECONDS = 300

SHOW_PREVIEW_WINDOW = os.getenv("SHOW_PREVIEW_WINDOW", "true").lower() == "true"
