package models

import "time"

type Camera struct {
	ID        string     `json:"id"`
	Codice    string     `json:"codice"`
	Nome      string     `json:"nome"`
	Posizione string     `json:"posizione,omitempty"`
	Stato     string     `json:"stato"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
}

type EventIn struct {
	EventID    string                 `json:"event_id"`
	Camera     string                 `json:"camera"`
	TipoEvento string                 `json:"tipo_evento"`
	Severity   string                 `json:"severity"`
	TrackID    string                 `json:"track_id"`
	Confidence float64                `json:"confidence"`
	BBox       *BBox                  `json:"bbox"`
	Zone       string                 `json:"zone,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type BBox struct {
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

type Event struct {
	ID         string                 `json:"id"`
	CameraID   string                 `json:"camera_id"`
	CameraNome string                 `json:"camera_nome"`
	TipoEvento string                 `json:"tipo_evento"`
	Severity   string                 `json:"severity"`
	TrackID    *string                `json:"track_id"`
	Confidence float64                `json:"confidence"`
	BBox       *BBox                  `json:"bbox,omitempty"`
	Zone       string                 `json:"zone,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	PersonID   *string                `json:"person_id,omitempty"`
}

type Person struct {
	ID         string    `json:"id"`
	TrackID    string    `json:"track_id"`
	Nome       string    `json:"nome"`
	IsCritical bool      `json:"is_critical"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}
