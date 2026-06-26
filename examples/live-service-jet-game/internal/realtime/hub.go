package realtime

import (
	"encoding/json"
	"fmt"
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
	mu         sync.RWMutex
	clients    map[*websocket.Conn]bool
	ships      map[string]shared.ShipState
	inputs     map[string]protocol.InputMessage
	bullets    map[string]shared.BulletState
	cooldowns  map[string]float64
	nextBullet int
}

func NewHub() *Hub {
	return &Hub{
		clients:   map[*websocket.Conn]bool{},
		ships:     map[string]shared.ShipState{},
		inputs:    map[string]protocol.InputMessage{},
		bullets:   map[string]shared.BulletState{},
		cooldowns: map[string]float64{},
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
		if input.X != 0 || input.Y != 0 {
			ship.X = input.X
			ship.Y = input.Y
		}
		if input.Angle != 0 {
			ship.Angle = input.Angle
		} else {
			ship.Angle += float64(input.Turn) * 3.2 * dt
		}
		if input.Thrust {
			ship.Speed += (26 + float64(input.SpeedLevel)*4) * dt
		}
		if input.Boost {
			ship.Speed += 18 * dt
		}
		ship.Speed *= 0.95
		ship.Speed = clamp(ship.Speed, 24, 120+float64(input.SpeedLevel)*12)
		ship.X += math.Cos(ship.Angle) * ship.Speed * dt
		ship.Y += math.Sin(ship.Angle) * ship.Speed * dt
		wrap(&ship.X, 960)
		wrap(&ship.Y, 640)
		ship.SpeedLevel = input.SpeedLevel
		ship.DamageLevel = input.DamageLevel
		ship.FireRateLevel = input.FireRateLevel
		h.ships[id] = ship
		h.cooldowns[id] -= dt
		if input.Shoot && h.cooldowns[id] <= 0 {
			h.spawnBullet(ship, input)
			h.cooldowns[id] = math.Max(0.1, 0.38-float64(ship.Level)*0.018-float64(input.FireRateLevel)*0.035)
		}
	}
	h.stepBullets(dt)
}

func (h *Hub) broadcastState() {
	h.mu.RLock()
	ships := make([]shared.ShipState, 0, len(h.ships))
	for _, ship := range h.ships {
		ships = append(ships, ship)
	}
	bullets := make([]shared.BulletState, 0, len(h.bullets))
	for _, bullet := range h.bullets {
		bullets = append(bullets, bullet)
	}
	sort.Slice(ships, func(i, j int) bool {
		return ships[i].PlayerID < ships[j].PlayerID
	})
	sort.Slice(bullets, func(i, j int) bool {
		return bullets[i].ID < bullets[j].ID
	})
	message := protocol.StateMessage{Type: "state", Ships: ships, Bullets: bullets}
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
	level := 1
	maxHP := 100
	return shared.ShipState{
		PlayerID: playerID,
		X:        160 + float64(index%5)*120,
		Y:        140 + float64(index/5)*100,
		Angle:    float64(index) * 0.8,
		Speed:    42,
		Alive:    true,
		Level:    level,
		HP:       maxHP,
		MaxHP:    maxHP,
	}
}

func (h *Hub) spawnBullet(ship shared.ShipState, input protocol.InputMessage) {
	h.nextBullet++
	speed := 260 + float64(ship.Level)*8
	damage := 18 + ship.Level*3 + input.DamageLevel*7
	id := fmt.Sprintf("b-%d", h.nextBullet)
	h.bullets[id] = shared.BulletState{
		ID:      id,
		OwnerID: ship.PlayerID,
		X:       ship.X + math.Cos(ship.Angle)*20,
		Y:       ship.Y + math.Sin(ship.Angle)*20,
		VX:      math.Cos(ship.Angle) * speed,
		VY:      math.Sin(ship.Angle) * speed,
		Damage:  damage,
	}
}

func (h *Hub) stepBullets(dt float64) {
	for id, bullet := range h.bullets {
		bullet.X += bullet.VX * dt
		bullet.Y += bullet.VY * dt
		if bullet.X < -30 || bullet.X > 990 || bullet.Y < -30 || bullet.Y > 670 {
			delete(h.bullets, id)
			continue
		}
		hit := ""
		for playerID, ship := range h.ships {
			if playerID == bullet.OwnerID || !ship.Alive {
				continue
			}
			if math.Hypot(ship.X-bullet.X, ship.Y-bullet.Y) <= 18 {
				hit = playerID
				break
			}
		}
		if hit == "" {
			h.bullets[id] = bullet
			continue
		}
		delete(h.bullets, id)
		target := h.ships[hit]
		target.HP -= bullet.Damage
		if target.HP <= 0 {
			killer := h.ships[bullet.OwnerID]
			killer.Score++
			killer.Level = 1 + killer.Score/2
			killer.MaxHP = 100 + killer.Level*12
			killer.HP = min(killer.MaxHP, killer.HP+35)
			h.ships[bullet.OwnerID] = killer
			target = spawnShip(hit, len(h.ships))
		}
		h.ships[hit] = target
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
