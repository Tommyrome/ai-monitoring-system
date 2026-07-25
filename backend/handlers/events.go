package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"aicam-backend/models"
	"aicam-backend/ws"
)

type EventsHandler struct {
	DB  *sql.DB
	Hub *ws.Hub
}

func NewEventsHandler(db *sql.DB, hub *ws.Hub) *EventsHandler {
	return &EventsHandler{DB: db, Hub: hub}
}

// CreateEvent - POST /api/events
// Chiamato dal modulo Python ogni volta che viene rilevata una persona
// davanti alla telecamera.
//
// Il backend e' l'unica fonte di verita' per il riconoscimento delle
// persone: qui si risolve (o si crea) la persona a partire dal track_id
// ricevuto dal tracker, e si decide se l'evento e' NORMAL o CRITICAL in
// base al flag is_critical attualmente salvato in database. In questo
// modo, se un operatore marca una persona come critica, il prossimo
// evento generato da quella persona sara' gia' correttamente CRITICAL,
// senza dover riavviare il modulo AI.
func (h *EventsHandler) CreateEvent(c *gin.Context) {
	var in models.EventIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload non valido: " + err.Error()})
		return
	}

	if in.Camera == "" || in.TrackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "camera e track_id sono obbligatori"})
		return
	}
	if in.Timestamp.IsZero() {
		in.Timestamp = time.Now()
	}

	// risolvi (o crea al volo) la telecamera dal suo "codice"
	var cameraID, cameraNome string
	err := h.DB.QueryRow(
		`SELECT id, nome FROM cameras WHERE codice = $1`, in.Camera,
	).Scan(&cameraID, &cameraNome)

	if err == sql.ErrNoRows {
		err = h.DB.QueryRow(
			`INSERT INTO cameras (codice, nome, stato, last_seen)
			 VALUES ($1, $1, 'online', now()) RETURNING id, nome`,
			in.Camera,
		).Scan(&cameraID, &cameraNome)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore risoluzione telecamera: " + err.Error()})
		return
	}
	_, _ = h.DB.Exec(`UPDATE cameras SET stato = 'online', last_seen = now() WHERE id = $1`, cameraID)

	// risolvi (o crea) la persona a partire dal track_id del tracker
	personID, personNome, isCritical, err := h.resolvePerson(in.TrackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore risoluzione persona: " + err.Error()})
		return
	}

	tipoEvento := "NORMAL"
	if isCritical {
		tipoEvento = "CRITICAL"
	}

	var metaJSON []byte
	if in.Metadata != nil {
		metaJSON, _ = json.Marshal(in.Metadata)
	}

	var newID string
	err = h.DB.QueryRow(
		`INSERT INTO events (camera_id, person_id, tipo_evento, track_id, confidence,
			bbox_x, bbox_y, bbox_w, bbox_h, metadata, "timestamp")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		cameraID, personID, tipoEvento, in.TrackID, in.Confidence,
		bboxField(in.BBox, "x1"), bboxField(in.BBox, "y1"), bboxField(in.BBox, "w"), bboxField(in.BBox, "h"),
		nullableJSON(metaJSON), in.Timestamp,
	).Scan(&newID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore salvataggio evento: " + err.Error()})
		return
	}

	trackID := in.TrackID
	out := models.Event{
		ID:         newID,
		CameraID:   cameraID,
		CameraNome: cameraNome,
		TipoEvento: tipoEvento,
		TrackID:    &trackID,
		Confidence: in.Confidence,
		BBox:       in.BBox,
		Metadata:   in.Metadata,
		Timestamp:  in.Timestamp,
		PersonID:   &personID,
		PersonNome: personNome,
	}

	// notifica in tempo reale tutti i client della dashboard connessi
	h.Hub.Broadcast(gin.H{"type": "new_event", "event": out})

	// invia anche le statistiche aggiornate, cosi' il pannello si aggiorna
	// senza dover ricaricare la pagina
	if stats, statsErr := h.computeStats(); statsErr == nil {
		h.Hub.Broadcast(gin.H{"type": "stats_update", "stats": stats})
	}

	c.JSON(http.StatusCreated, out)
}

// resolvePerson trova la persona associata al track_id, oppure la crea se
// e' la prima volta che viene vista. Ritorna id, nome e stato is_critical
// correnti (letti dal database, non dalla cache del modulo AI).
func (h *EventsHandler) resolvePerson(trackID string) (id string, nome string, isCritical bool, err error) {
	err = h.DB.QueryRow(
		`SELECT id, nome, is_critical FROM persons WHERE track_id = $1`, trackID,
	).Scan(&id, &nome, &isCritical)

	if err == sql.ErrNoRows {
		defaultNome := "Persona " + trackID
		err = h.DB.QueryRow(
			`INSERT INTO persons (track_id, nome, is_critical, first_seen, last_seen)
			 VALUES ($1, $2, FALSE, now(), now())
			 RETURNING id, nome, is_critical`,
			trackID, defaultNome,
		).Scan(&id, &nome, &isCritical)
		return id, nome, isCritical, err
	}
	if err != nil {
		return "", "", false, err
	}

	_, err = h.DB.Exec(`UPDATE persons SET last_seen = now() WHERE id = $1`, id)
	return id, nome, isCritical, err
}

// ListEvents - GET /api/events?limit=50&tipo=CRITICAL
func (h *EventsHandler) ListEvents(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	tipo := c.Query("tipo")

	query := `SELECT e.id, e.camera_id, c.nome, e.tipo_evento, e.track_id,
	                 e.confidence, e.bbox_x, e.bbox_y, e.bbox_w, e.bbox_h, e.metadata, e."timestamp",
	                 e.person_id, COALESCE(p.nome, '')
	          FROM events e
	          JOIN cameras c ON c.id = e.camera_id
	          LEFT JOIN persons p ON p.id = e.person_id`
	args := []interface{}{}
	if tipo != "" {
		query += ` WHERE e.tipo_evento = $1`
		args = append(args, tipo)
	}
	query += ` ORDER BY e."timestamp" DESC LIMIT ` + limit

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	events := make([]models.Event, 0)
	for rows.Next() {
		var e models.Event
		var trackID sql.NullString
		var bx, by, bw, bh, conf sql.NullFloat64
		var metaRaw []byte
		var personID sql.NullString

		if err := rows.Scan(&e.ID, &e.CameraID, &e.CameraNome, &e.TipoEvento,
			&trackID, &conf, &bx, &by, &bw, &bh, &metaRaw, &e.Timestamp,
			&personID, &e.PersonNome); err != nil {
			continue
		}
		if trackID.Valid {
			e.TrackID = &trackID.String
		}
		if conf.Valid {
			e.Confidence = conf.Float64
		}
		if bx.Valid && by.Valid && bw.Valid && bh.Valid {
			e.BBox = &models.BBox{X1: bx.Float64, Y1: by.Float64, X2: bx.Float64 + bw.Float64, Y2: by.Float64 + bh.Float64}
		}
		if personID.Valid {
			e.PersonID = &personID.String
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &e.Metadata)
		}
		events = append(events, e)
	}

	c.JSON(http.StatusOK, events)
}

// GetStats - GET /api/stats
func (h *EventsHandler) GetStats(c *gin.Context) {
	stats, err := h.computeStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *EventsHandler) computeStats() (models.Stats, error) {
	var stats models.Stats
	err := h.DB.QueryRow(
		`SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE tipo_evento = 'NORMAL'),
			COUNT(*) FILTER (WHERE tipo_evento = 'CRITICAL')
		 FROM events`,
	).Scan(&stats.TotalEvents, &stats.NormalEvents, &stats.CriticalEvents)
	return stats, err
}

func bboxField(b *models.BBox, field string) interface{} {
	if b == nil {
		return nil
	}
	switch field {
	case "x1":
		return b.X1
	case "y1":
		return b.Y1
	case "w":
		return b.X2 - b.X1
	case "h":
		return b.Y2 - b.Y1
	}
	return nil
}

func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
