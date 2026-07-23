package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"aicam-backend/models"
)

type CamerasHandler struct {
	DB *sql.DB
}

func NewCamerasHandler(db *sql.DB) *CamerasHandler {
	return &CamerasHandler{DB: db}
}

// ListCameras - GET /api/cameras
func (h *CamerasHandler) ListCameras(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, codice, nome, posizione, stato, last_seen FROM cameras ORDER BY nome`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	cameras := make([]models.Camera, 0)
	for rows.Next() {
		var cam models.Camera
		var posizione sql.NullString
		var lastSeen sql.NullTime
		if err := rows.Scan(&cam.ID, &cam.Codice, &cam.Nome, &posizione, &cam.Stato, &lastSeen); err != nil {
			continue
		}
		cam.Posizione = posizione.String
		if lastSeen.Valid {
			cam.LastSeen = &lastSeen.Time
		}
		cameras = append(cameras, cam)
	}

	c.JSON(http.StatusOK, cameras)
}
