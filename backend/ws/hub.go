package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub mantiene la lista dei client WebSocket connessi e permette il broadcast.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In produzione limitare l'origin agli host consentiti.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS gestisce l'upgrade della connessione HTTP a WebSocket.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("errore upgrade websocket:", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	log.Println("nuovo client dashboard connesso via websocket")

	// loop di lettura solo per rilevare la disconnessione del client
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			log.Println("client dashboard disconnesso")
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

// Broadcast invia un payload JSON a tutti i client connessi.
func (h *Hub) Broadcast(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("errore serializzazione broadcast:", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
