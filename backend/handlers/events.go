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
// o generata un'anomalia/alert di zona.
func (h *EventsHandler) CreateEvent(c *gin.Context) {
	var in models.EventIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload non valido: " + err.Error()})
		return
	}

	if in.Camera == "" || in.TipoEvento == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "camera e tipo_evento sono obbligatori"})
		return
	}
	if in.Severity == "" {
		in.Severity = "info"
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

	// aggiorna stato/last_seen della telecamera (eventi = telecamera online)
	_, _ = h.DB.Exec(`UPDATE cameras SET stato = 'online', last_seen = now() WHERE id = $1`, cameraID)

	var bboxJSON []byte
	if in.BBox != nil {
		bboxJSON, _ = json.Marshal(in.BBox)
	}
	var metaJSON []byte
	if in.Metadata != nil {
		metaJSON, _ = json.Marshal(in.Metadata)
	}

	var newID string
	err = h.DB.QueryRow(
		`INSERT INTO events (camera_id, tipo_evento, severity, track_id, confidence,
			bbox_x, bbox_y, bbox_w, bbox_h, metadata, "timestamp")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		cameraID, in.TipoEvento, in.Severity, in.TrackID, in.Confidence,
		bboxField(in.BBox, "x1"), bboxField(in.BBox, "y1"), bboxField(in.BBox, "w"), bboxField(in.BBox, "h"),
		nullableJSON(metaJSON), in.Timestamp,
	).Scan(&newID)

	_ = bboxJSON // (bbox completo disponibile via metadata/bbox_* se servisse in futuro)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "errore salvataggio evento: " + err.Error()})
		return
	}

	out := models.Event{
		ID:         newID,
		CameraID:   cameraID,
		CameraNome: cameraNome,
		TipoEvento: in.TipoEvento,
		Severity:   in.Severity,
		TrackID:    in.TrackID,
		Confidence: in.Confidence,
		BBox:       in.BBox,
		Zone:       in.Zone,
		Metadata:   in.Metadata,
		Timestamp:  in.Timestamp,
	}

	// notifica in tempo reale tutti i client della dashboard connessi
	h.Hub.Broadcast(gin.H{"type": "new_event", "event": out})

	c.JSON(http.StatusCreated, out)
}

// ListEvents - GET /api/events?limit=50&severity=critical
func (h *EventsHandler) ListEvents(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	severity := c.Query("severity")

	query := `SELECT e.id, e.camera_id, c.nome, e.tipo_evento, e.severity, e.track_id,
	                 e.confidence, e.bbox_x, e.bbox_y, e.bbox_w, e.bbox_h, e.metadata, e."timestamp"
	          FROM events e JOIN cameras c ON c.id = e.camera_id`
	args := []interface{}{}
	if severity != "" {
		query += ` WHERE e.severity = $1`
		args = append(args, severity)
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
		var bx, by, bw, bh sql.NullFloat64
		var metaRaw []byte

		if err := rows.Scan(&e.ID, &e.CameraID, &e.CameraNome, &e.TipoEvento, &e.Severity,
			&trackID, &e.Confidence, &bx, &by, &bw, &bh, &metaRaw, &e.Timestamp); err != nil {
			continue
		}
		if trackID.Valid {
			e.TrackID = &trackID.String
		}
		if bx.Valid && by.Valid && bw.Valid && bh.Valid {
			e.BBox = &models.BBox{X1: bx.Float64, Y1: by.Float64, X2: bx.Float64 + bw.Float64, Y2: by.Float64 + bh.Float64}
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &e.Metadata)
		}
		events = append(events, e)
	}

	c.JSON(http.StatusOK, events)
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
