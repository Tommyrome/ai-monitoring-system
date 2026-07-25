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

// EventIn e' il payload inviato dal modulo AI Python per ogni rilevamento.
// Il campo TipoEvento, se presente, viene ignorato: e' il backend a
// determinare in modo autoritativo se l'evento e' NORMAL o CRITICAL,
// leggendo lo stato is_critical della persona associata al track_id.
type EventIn struct {
	EventID    string                 `json:"event_id"`
	Camera     string                 `json:"camera"`
	TipoEvento string                 `json:"tipo_evento"`
	TrackID    string                 `json:"track_id"`
	Confidence float64                `json:"confidence"`
	BBox       *BBox                  `json:"bbox"`
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
	TipoEvento string                 `json:"tipo_evento"` // NORMAL | CRITICAL
	TrackID    *string                `json:"track_id"`
	Confidence float64                `json:"confidence"`
	BBox       *BBox                  `json:"bbox,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	PersonID   *string                `json:"person_id,omitempty"`
	PersonNome string                 `json:"person_nome,omitempty"`
}

type Person struct {
	ID         string    `json:"id"`
	TrackID    string    `json:"track_id"`
	Nome       string    `json:"nome"`
	IsCritical bool      `json:"is_critical"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}

// Stats e' il riepilogo aggregato mostrato nel pannello statistiche.
type Stats struct {
	TotalEvents    int `json:"total_events"`
	NormalEvents   int `json:"normal_events"`
	CriticalEvents int `json:"critical_events"`
}
