package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"example.com/orbit-snake/internal/protocol"
	"example.com/orbit-snake/internal/shared"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
	ships   map[string]shared.ShipState
}

func NewHub() *Hub {
	return &Hub{
		clients: map[*websocket.Conn]bool{},
		ships:   map[string]shared.ShipState{},
	}
}

func (h *Hub) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"realtime"}`))
	})
	mux.HandleFunc("GET /ws", h.websocket)
	return mux
}

func (h *Hub) Run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		h.broadcastState()
	}
}

func (h *Hub) websocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	go h.readLoop(conn)
}

func (h *Hub) readLoop(conn *websocket.Conn) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		_ = conn.Close()
	}()
	for {
		var input protocol.InputMessage
		if err := conn.ReadJSON(&input); err != nil {
			return
		}
		if input.PlayerID == "" {
			continue
		}
		h.mu.Lock()
		ship := h.ships[input.PlayerID]
		ship.PlayerID = input.PlayerID
		ship.Alive = true
		if input.Thrust {
			ship.Speed += 0.2
		}
		ship.Angle += float64(input.Turn) * 0.1
		h.ships[input.PlayerID] = ship
		h.mu.Unlock()
	}
}

func (h *Hub) broadcastState() {
	h.mu.RLock()
	ships := make([]shared.ShipState, 0, len(h.ships))
	for _, ship := range h.ships {
		ships = append(ships, ship)
	}
	message := protocol.StateMessage{Type: "state", Ships: ships}
	data, _ := json.Marshal(message)
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.RUnlock()

	for _, conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			_ = conn.Close()
		}
	}
}
