package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"aicam-backend/ws"
)

// FrameHandler mantiene in memoria l'ultimo frame JPEG (base64) inviato dal
// modulo AI per ciascuna telecamera, cosi' la dashboard puo' mostrare un
// feed "live" senza bisogno di un vero server di streaming video: il
// modulo Python invia un frame ogni pochi millisecondi via HTTP, il
// backend lo tiene in cache e lo ridistribuisce ai client via WebSocket.
type FrameHandler struct {
	Hub *ws.Hub

	mu     sync.RWMutex
	latest map[string]frameEntry
}

type frameEntry struct {
	Data      string    `json:"data"` // JPEG codificato in base64
	Timestamp time.Time `json:"timestamp"`
}

func NewFrameHandler(hub *ws.Hub) *FrameHandler {
	return &FrameHandler{Hub: hub, latest: make(map[string]frameEntry)}
}

type frameInput struct {
	Camera    string `json:"camera" binding:"required"`
	Data      string `json:"data" binding:"required"` // JPEG base64, senza prefisso data:URL
	Timestamp string `json:"timestamp"`
}

// PostFrame - POST /api/frame (chiamato dal modulo AI)
func (h *FrameHandler) PostFrame(c *gin.Context) {
	var in frameInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload non valido: " + err.Error()})
		return
	}

	entry := frameEntry{Data: in.Data, Timestamp: time.Now()}

	h.mu.Lock()
	h.latest[in.Camera] = entry
	h.mu.Unlock()

	h.Hub.Broadcast(gin.H{
		"type":      "frame",
		"camera":    in.Camera,
		"data":      entry.Data,
		"timestamp": entry.Timestamp,
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetFrame - GET /api/frame?camera=camera_01 (usato dalla dashboard al
// primo caricamento, prima che arrivi il primo messaggio WebSocket)
func (h *FrameHandler) GetFrame(c *gin.Context) {
	camera := c.DefaultQuery("camera", "camera_01")

	h.mu.RLock()
	entry, ok := h.latest[camera]
	h.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "nessun frame disponibile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"camera": camera, "data": entry.Data, "timestamp": entry.Timestamp})
}
