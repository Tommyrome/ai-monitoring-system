import os
from dotenv import load_dotenv

load_dotenv()

# --- Backend ---
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")
EVENTS_ENDPOINT = f"{BACKEND_URL}/api/events"
API_TOKEN = os.getenv("AI_SERVICE_TOKEN", "change-me-shared-secret")

# --- Camera ---
CAMERA_SOURCE = os.getenv("CAMERA_SOURCE", "0")
CAMERA_CODE = os.getenv("CAMERA_CODE", "camera_01")

# --- Modello YOLO ---
MODEL_WEIGHTS = os.getenv("MODEL_WEIGHTS", "yolov8n.pt")
CONFIDENCE_THRESHOLD = float(os.getenv("CONFIDENCE_THRESHOLD", "0.5"))
PERSON_CLASS_ID = 0

# --- Frame processing ---
PROCESS_EVERY_N_FRAMES = int(os.getenv("PROCESS_EVERY_N_FRAMES", "2"))
SEND_MIN_INTERVAL_SECONDS = float(os.getenv("SEND_MIN_INTERVAL_SECONDS", "1.5"))  # throttling più conservativo

# --- Zone ---
ZONES_CONFIG = []  # Rimosse tutte le zone vietate

# --- Anomaly detection (solo orari e overcrowding) ---
ALLOWED_HOUR_START = "06:00"
ALLOWED_HOUR_END = "22:00"
MAX_PEOPLE_PER_ZONE = 5
MAX_DWELL_SECONDS = 300

SHOW_PREVIEW_WINDOW = os.getenv("SHOW_PREVIEW_WINDOW", "true").lower() == "true"
