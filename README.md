# AI Camera Monitoring System

Sistema di monitoraggio intelligente basato su Computer Vision.

Rileva automaticamente la presenza di persone tramite webcam, genera eventi
strutturati (rilevamento, accesso a zona vietata, anomalie) e li mostra in
tempo reale su una dashboard web — senza fare riconoscimento facciale: le
persone restano anonime, il sistema traccia solo "un oggetto di tipo person"
con un ID temporaneo.

## Architettura

```
┌──────────────────┐      HTTP POST /api/events      ┌──────────────────┐
│   AI SERVICE      │ ───────────────────────────────▶│     BACKEND       │
│   (Python)        │                                  │     (Go)          │
│                    │                                  │                    │
│ OpenCV + YOLOv8    │                                  │ Gin REST API       │
│ CentroidTracker    │                                  │ PostgreSQL         │
│ Zone / Anomaly     │                                  │ WebSocket Hub      │
└──────────────────┘                                  └────────┬─────────┘
                                                                  │ WebSocket
                                                                  │ (broadcast)
                                                        ┌─────────▼─────────┐
                                                        │   DASHBOARD        │
                                                        │   (HTML/CSS/JS)    │
                                                        │   fetch + WS live  │
                                                        └────────────────────┘
```

Tre servizi indipendenti, containerizzati separatamente:

| Servizio      | Linguaggio | Responsabilità |
|---------------|-----------|----------------|
| `ai-service`  | Python     | Cattura video, object detection, tracking, invio eventi |
| `backend`     | Go         | API REST, autenticazione JWT, persistenza, WebSocket |
| `frontend`    | HTML/CSS/JS| Dashboard live: telecamere, log eventi, statistiche |

## Struttura del progetto

```
ai-monitoring-system/
├── ai-service/          # Python: OpenCV + YOLO + tracking
│   ├── detector.py       # loop principale
│   ├── tracker.py        # CentroidTracker
│   ├── zones.py          # zone monitoring + anomaly detection
│   ├── event_client.py   # invio eventi HTTP al backend
│   ├── config.py
│   ├── requirements.txt
│   └── Dockerfile
├── backend/              # Go: API REST + WebSocket
│   ├── main.go
│   ├── db.go
│   ├── handlers/          (events, cameras, auth)
│   ├── middleware/         (JWT + service token)
│   ├── ws/                 (hub WebSocket)
│   ├── models/
│   ├── go.mod
│   └── Dockerfile
├── frontend/              # Dashboard statica
│   ├── index.html
│   ├── style.css
│   └── app.js
├── db/
│   └── schema.sql         # tabelle users, cameras, zones, events
├── docker-compose.yml
└── README.md
```

## Come avviarlo

### 1. Avvia backend e database

```bash
docker compose up -d postgres backend
```

Questo avvia PostgreSQL (con schema già inizializzato) e il backend Go su
`http://localhost:8080`.

### 2. Crea un utente admin

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### 3. Avvia il modulo AI (sull'host, per usare la webcam)

L'accesso alla webcam da dentro Docker è scomodo (soprattutto su
macOS/Windows), quindi conviene eseguire il modulo Python direttamente
sull'host:

```bash
cd ai-service
python -m venv venv
source venv/bin/activate        # su Windows: venv\Scripts\activate
pip install -r requirements.txt
cp .env.example .env             # e adatta BACKEND_URL se serve
python detector.py
```

Al primo avvio `ultralytics` scarica automaticamente i pesi `yolov8n.pt`
(richiede una connessione internet la prima volta).

### 4. Avvia il frontend

La dashboard è statica: basta aprirla con un server locale qualsiasi, ad
esempio:

```bash
cd frontend
python -m http.server 5500
```

Poi vai su `http://localhost:5500`, accedi con `admin / admin123` e vedrai
gli eventi arrivare in tempo reale non appena il modulo AI rileva una
persona davanti alla webcam.

## Variabili d'ambiente principali

| Variabile | Servizio | Descrizione |
|---|---|---|
| `DATABASE_URL` | backend | connection string PostgreSQL |
| `JWT_SECRET` | backend | chiave di firma dei JWT utente |
| `AI_SERVICE_TOKEN` | backend + ai-service | token condiviso per l'endpoint `POST /api/events` |
| `BACKEND_URL` | ai-service | URL del backend Go |
| `CAMERA_SOURCE` | ai-service | `0` per la webcam di default, oppure path/URL RTSP |
| `CONFIDENCE_THRESHOLD` | ai-service | soglia minima di confidenza YOLO (default 0.5) |
