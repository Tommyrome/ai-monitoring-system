# AI Camera Monitoring System

Sistema di monitoraggio intelligente con una singola telecamera, basato
su Computer Vision.

Rileva automaticamente le persone tramite webcam, le riconosce nel tempo
tramite un identificativo persistente (tracking per centroide — non
riconoscimento facciale), permette di rinominarle e di contrassegnarle
come "critiche". Ogni evento generato viene classificato come `NORMAL`
o `CRITICAL` in base allo stato della persona, mostrato in tempo reale
su una dashboard web insieme al feed della telecamera, alle statistiche
e al log eventi.

## Architettura

```
┌──────────────────┐   POST /api/events, /api/frame   ┌──────────────────┐
│   AI SERVICE      │ ───────────────────────────────▶│     BACKEND       │
│   (Python)        │                                  │     (Go)          │
│                    │                                  │                    │
│ OpenCV + YOLOv8    │◀──── GET /api/persons ──────────│ Gin REST API       │
│ CentroidTracker    │      (cache locale nome/critica) │ PostgreSQL         │
└──────────────────┘                                  │ WebSocket Hub      │
                                                        └────────┬─────────┘
                                                                  │ WebSocket
                                                                  │ (eventi, statistiche, frame)
                                                        ┌─────────▼─────────┐
                                                        │   DASHBOARD        │
                                                        │   (HTML/CSS/JS)    │
                                                        │   feed + log + stat│
                                                        └────────────────────┘
```

Tre servizi indipendenti, containerizzati separatamente:

| Servizio      | Linguaggio | Responsabilità |
|---------------|-----------|----------------|
| `ai-service`  | Python     | Cattura video, object detection, tracking, invio eventi e frame |
| `backend`     | Go         | API REST, autenticazione JWT, riconoscimento/persistenza persone, WebSocket |
| `frontend`    | HTML/CSS/JS| Dashboard live: feed telecamera, log eventi, statistiche, persone |

Il backend è l'unica fonte di verità su nome e stato "critica" di ogni
persona: quando arriva un evento, risolve (o crea) la persona dal
`track_id` e decide se l'evento è `NORMAL` o `CRITICAL` leggendo lo
stato attuale dal database — così una rinomina o un cambio di stato
fatti dalla dashboard hanno effetto immediato, senza riavviare il
modulo AI.

## Struttura del progetto

```
ai-monitoring-system/
├── ai-service/          # Python: OpenCV + YOLO + tracking
│   ├── detector.py       # loop principale + streaming frame
│   ├── tracker.py        # CentroidTracker
│   ├── event_client.py   # invio eventi HTTP al backend
│   ├── config.py
│   ├── requirements.txt
│   └── Dockerfile
├── backend/              # Go: API REST + WebSocket
│   ├── main.go
│   ├── db.go
│   ├── handlers/          (events, persons, frame, cameras, auth)
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
│   └── schema.sql         # tabelle users, cameras, persons, events
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
il feed della telecamera, gli eventi e le persone rilevate aggiornarsi in
tempo reale non appena il modulo AI rileva qualcuno davanti alla webcam.

## API principali

| Metodo | Endpoint | Autenticazione | Descrizione |
|---|---|---|---|
| `POST` | `/api/auth/register` / `/api/auth/login` | — | crea utente / ottiene un JWT |
| `POST` | `/api/events` | service token | riceve un rilevamento dal modulo AI, risolve la persona e classifica l'evento |
| `POST` | `/api/frame` | service token | riceve un frame JPEG (base64) dal modulo AI e lo ridistribuisce via WebSocket |
| `GET` | `/api/events` | JWT | elenco log eventi (filtro opzionale `?tipo=CRITICAL`) |
| `GET` | `/api/stats` | JWT | conteggio eventi totali / normali / critici |
| `GET` | `/api/persons` | JWT | elenco persone note |
| `PATCH` | `/api/persons/:id` | JWT | rinomina e/o imposta lo stato "critica" (`{"nome": "...", "is_critical": true}`) |
| `GET` | `/api/frame` | JWT | ultimo frame disponibile (usato al primo caricamento) |
| `GET` | `/ws` | — | canale WebSocket: messaggi `new_event`, `stats_update`, `frame` |

## Variabili d'ambiente principali

| Variabile | Servizio | Descrizione |
|---|---|---|
| `DATABASE_URL` | backend | connection string PostgreSQL |
| `JWT_SECRET` | backend | chiave di firma dei JWT utente |
| `AI_SERVICE_TOKEN` | backend + ai-service | token condiviso per gli endpoint `POST /api/events` e `POST /api/frame` |
| `BACKEND_URL` | ai-service | URL del backend Go |
| `CAMERA_SOURCE` | ai-service | `0` per la webcam di default, oppure path/URL RTSP |
| `CONFIDENCE_THRESHOLD` | ai-service | soglia minima di confidenza YOLO (default 0.5) |
| `KNOWN_PERSONS_REFRESH_SECONDS` | ai-service | ogni quanto ricaricare nome/stato critica dal backend (default 5s) |
| `FRAME_STREAM_MIN_INTERVAL` | ai-service | intervallo minimo tra due frame inviati alla dashboard (default 0.15s) |
| `FRAME_JPEG_QUALITY` | ai-service | qualità JPEG del feed inviato alla dashboard (default 70) |
