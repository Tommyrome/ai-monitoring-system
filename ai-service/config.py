import os
from dotenv import load_dotenv

load_dotenv()

# --- Backend ---
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")
EVENTS_ENDPOINT = f"{BACKEND_URL}/api/events"
FRAME_ENDPOINT = f"{BACKEND_URL}/api/frame"
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
NEW_APPEARANCE_GAP_SECONDS = float(os.getenv("NEW_APPEARANCE_GAP_SECONDS", "3.0"))

# --- Streaming del feed verso la dashboard ---
FRAME_STREAM_MIN_INTERVAL = float(os.getenv("FRAME_STREAM_MIN_INTERVAL", "0.15"))  # ~6-7 fps
FRAME_JPEG_QUALITY = int(os.getenv("FRAME_JPEG_QUALITY", "70"))

# --- Persone note: ogni quanto ricaricare nome/is_critical dal backend,
#     cosi' una rinomina o un cambio di stato "critica" fatti dalla
#     dashboard vengono recepiti senza riavviare il modulo AI ---
KNOWN_PERSONS_REFRESH_SECONDS = float(os.getenv("KNOWN_PERSONS_REFRESH_SECONDS", "5.0"))

SHOW_PREVIEW_WINDOW = os.getenv("SHOW_PREVIEW_WINDOW", "true").lower() == "true"
