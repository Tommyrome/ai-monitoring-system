package models

import "time"

// Camera rappresenta una telecamera registrata nel sistema.
type Camera struct {
	ID        string     `json:"id"`
	Codice    string     `json:"codice"`
	Nome      string     `json:"nome"`
	Posizione string     `json:"posizione"`
	Stato     string     `json:"stato"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
}

// BBox rappresenta il bounding box normalizzato in pixel.
type BBox struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

// EventIn e' il payload ricevuto dal modulo Python (POST /api/events).
type EventIn struct {
	EventID    string                 `json:"event_id"`
	Camera     string                 `json:"camera"`     // codice telecamera, es. "camera_01"
	TipoEvento string                 `json:"tipo_evento"` // person_detected | zone_alert | anomaly
	Severity   string                 `json:"severity"`    // info | warning | critical
	TrackID    *string                `json:"track_id"`
	Confidence float64                `json:"confidence"`
	BBox       *BBox                  `json:"bbox"`
	Zone       *string                `json:"zone"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Event e' la rappresentazione persistita/restituita dal backend.
type Event struct {
	ID         string                 `json:"id"`
	CameraID   string                 `json:"camera_id"`
	CameraNome string                 `json:"camera_nome,omitempty"`
	TipoEvento string                 `json:"tipo_evento"`
	Severity   string                 `json:"severity"`
	TrackID    *string                `json:"track_id,omitempty"`
	Confidence float64                `json:"confidence"`
	BBox       *BBox                  `json:"bbox,omitempty"`
	Zone       *string                `json:"zone,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// User rappresenta un utente della dashboard.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Ruolo        string `json:"ruolo"`
}
