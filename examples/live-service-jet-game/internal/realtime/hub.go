package realtime

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sort"
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
	inputs  map[string]protocol.InputMessage
}

func NewHub() *Hub {
	return &Hub{
		clients: map[*websocket.Conn]bool{},
		ships:   map[string]shared.ShipState{},
		inputs:  map[string]protocol.InputMessage{},
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
		h.step(0.1)
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
		if _, ok := h.ships[input.PlayerID]; !ok {
			h.ships[input.PlayerID] = spawnShip(input.PlayerID, len(h.ships))
		}
		h.inputs[input.PlayerID] = input
		h.mu.Unlock()
	}
}

func (h *Hub) step(dt float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, ship := range h.ships {
		input := h.inputs[id]
		ship.Alive = true
		ship.Angle += float64(input.Turn) * 3.2 * dt
		if input.Thrust {
			ship.Speed += 26 * dt
		}
		if input.Boost {
			ship.Speed += 18 * dt
		}
		ship.Speed *= 0.95
		ship.Speed = clamp(ship.Speed, 24, 120)
		ship.X += math.Cos(ship.Angle) * ship.Speed * dt
		ship.Y += math.Sin(ship.Angle) * ship.Speed * dt
		wrap(&ship.X, 960)
		wrap(&ship.Y, 640)
		h.ships[id] = ship
	}
}

func (h *Hub) broadcastState() {
	h.mu.RLock()
	ships := make([]shared.ShipState, 0, len(h.ships))
	for _, ship := range h.ships {
		ships = append(ships, ship)
	}
	sort.Slice(ships, func(i, j int) bool {
		return ships[i].PlayerID < ships[j].PlayerID
	})
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

func spawnShip(playerID string, index int) shared.ShipState {
	return shared.ShipState{
		PlayerID: playerID,
		X:        160 + float64(index%5)*120,
		Y:        140 + float64(index/5)*100,
		Angle:    float64(index) * 0.8,
		Speed:    42,
		Alive:    true,
	}
}

func wrap(v *float64, max float64) {
	if *v < 0 {
		*v += max
	}
	if *v > max {
		*v -= max
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
