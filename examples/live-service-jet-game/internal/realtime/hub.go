package realtime

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"example.com/orbit-snake/internal/protocol"
	"example.com/orbit-snake/internal/shared"
)

type Hub struct {
	mu          sync.RWMutex
	clients     map[*websocket.Conn]bool
	clientIDs   map[*websocket.Conn]string
	ships       map[string]shared.ShipState
	inputs      map[string]protocol.InputMessage
	crystals    map[string]shared.Crystal
	bullets     map[string]shared.BulletState
	cooldowns   map[string]float64
	nextCrystal int
	nextBullet  int
}

func NewHub() *Hub {
	h := &Hub{
		clients:   map[*websocket.Conn]bool{},
		clientIDs: map[*websocket.Conn]string{},
		ships:     map[string]shared.ShipState{},
		inputs:    map[string]protocol.InputMessage{},
		crystals:  map[string]shared.Crystal{},
		bullets:   map[string]shared.BulletState{},
		cooldowns: map[string]float64{},
	}
	for len(h.crystals) < 36 {
		h.spawnCrystalLocked()
	}
	return h
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
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		h.step(0.05)
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
	log.Printf("realtime websocket connected clients=%d", len(h.clients))
	h.mu.Unlock()
	go h.readLoop(conn)
}

func (h *Hub) readLoop(conn *websocket.Conn) {
	defer func() {
		h.mu.Lock()
		if playerID := h.clientIDs[conn]; playerID != "" {
			delete(h.ships, playerID)
			delete(h.inputs, playerID)
			delete(h.cooldowns, playerID)
			log.Printf("player disconnected id=%s remaining=%d", playerID, len(h.clients)-1)
		}
		delete(h.clients, conn)
		delete(h.clientIDs, conn)
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
		if existing := h.clientIDs[conn]; existing != "" && existing != input.PlayerID {
			delete(h.ships, existing)
			delete(h.inputs, existing)
			delete(h.cooldowns, existing)
		}
		h.clientIDs[conn] = input.PlayerID
		if _, ok := h.ships[input.PlayerID]; !ok {
			h.ships[input.PlayerID] = spawnShip(input.PlayerID, input.PlayerName, len(h.ships))
			log.Printf("player joined id=%s name=%q", input.PlayerID, input.PlayerName)
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
		before := ship
		respawned := false
		if input.PlayerName != "" {
			ship.PlayerName = input.PlayerName
		}
		if input.Respawn {
			ship = spawnShip(id, ship.PlayerName, len(h.ships))
			respawned = true
			input.Respawn = false
			h.inputs[id] = input
			log.Printf("player respawned id=%s name=%q level=%d xp=%d points=%d", id, ship.PlayerName, ship.Level, ship.XP, ship.UpgradePoints)
		}
		if !ship.Alive {
			h.ships[id] = ship
			continue
		}
		if input.X != 0 || input.Y != 0 {
			ship.X = input.X
			ship.Y = input.Y
		}
		if input.Angle != 0 {
			ship.Angle = input.Angle
		} else {
			ship.Angle += float64(input.Turn) * 3.2 * dt
		}
		ship.SpeedLevel = input.SpeedLevel
		ship.TurnLevel = input.TurnLevel
		ship.DamageLevel = input.DamageLevel
		ship.FireRateLevel = input.FireRateLevel
		ship.UpgradePoints = input.UpgradePoints
		h.collectCrystalsLocked(&ship, input.CollectedIDs)
		if ship.Level < before.Level {
			log.Printf("level decreased id=%s before=%d after=%d respawn=%v alive=%v", id, before.Level, ship.Level, respawned, ship.Alive)
		}
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
	crystals := make([]shared.Crystal, 0, len(h.crystals))
	for _, crystal := range h.crystals {
		crystals = append(crystals, crystal)
	}
	sort.Slice(ships, func(i, j int) bool {
		return ships[i].PlayerID < ships[j].PlayerID
	})
	sort.Slice(bullets, func(i, j int) bool {
		return bullets[i].ID < bullets[j].ID
	})
	sort.Slice(crystals, func(i, j int) bool {
		return crystals[i].ID < crystals[j].ID
	})
	message := protocol.StateMessage{Type: "state", Ships: ships, Crystals: crystals, Bullets: bullets}
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

func (h *Hub) collectCrystalsLocked(ship *shared.ShipState, ids []string) {
	for _, id := range ids {
		crystal, ok := h.crystals[id]
		if !ok {
			log.Printf("crystal collect ignored id=%s player=%s reason=missing", id, ship.PlayerID)
			continue
		}
		dist := math.Hypot(ship.X-crystal.X, ship.Y-crystal.Y)
		if dist > 28 {
			log.Printf("crystal collect rejected id=%s player=%s dist=%.1f", id, ship.PlayerID, dist)
			continue
		}
		beforeLevel := ship.Level
		beforeXP := ship.XP
		ship.Score += crystal.Value
		addXP(ship, crystal.Value)
		log.Printf("crystal collected id=%s player=%s value=%d xp=%d->%d level=%d->%d points=%d", id, ship.PlayerID, crystal.Value, beforeXP, ship.XP, beforeLevel, ship.Level, ship.UpgradePoints)
		delete(h.crystals, id)
		h.spawnCrystalLocked()
	}
}

func (h *Hub) spawnCrystalLocked() {
	h.nextCrystal++
	value := 1 + rand.Intn(3)
	rarity := "common"
	if value == 2 {
		rarity = "bright"
	} else if value == 3 {
		rarity = "rare"
	}
	id := fmt.Sprintf("c-%d", h.nextCrystal)
	h.crystals[id] = shared.Crystal{
		ID:     id,
		X:      32 + rand.Float64()*(960-64),
		Y:      32 + rand.Float64()*(640-64),
		Value:  value,
		Rarity: rarity,
	}
}

func spawnShip(playerID, playerName string, index int) shared.ShipState {
	if playerName == "" {
		playerName = playerID
	}
	level := 0
	maxHP := 100
	return shared.ShipState{
		PlayerID:   playerID,
		PlayerName: playerName,
		X:          160 + float64(index%5)*120,
		Y:          140 + float64(index/5)*100,
		Angle:      float64(index) * 0.8,
		Speed:      42,
		Alive:      true,
		Level:      level,
		NextXP:     nextXP(level),
		HP:         maxHP,
		MaxHP:      maxHP,
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
			killerBeforeLevel := killer.Level
			killer.Score++
			addXP(&killer, 8)
			killer.MaxHP = 100 + killer.Level*12
			killer.HP = min(killer.MaxHP, killer.HP+35)
			h.ships[bullet.OwnerID] = killer
			target.Alive = false
			target.HP = 0
			log.Printf("ship destroyed victim=%s killer=%s killer_level=%d->%d killer_xp=%d points=%d", hit, bullet.OwnerID, killerBeforeLevel, killer.Level, killer.XP, killer.UpgradePoints)
		}
		h.ships[hit] = target
	}
}

func addXP(ship *shared.ShipState, amount int) {
	ship.XP += amount
	for ship.XP >= nextXP(ship.Level) {
		ship.XP -= nextXP(ship.Level)
		ship.Level++
		ship.UpgradePoints++
		ship.NextXP = nextXP(ship.Level)
		log.Printf("level up player=%s level=%d xp=%d next=%d points=%d", ship.PlayerID, ship.Level, ship.XP, ship.NextXP, ship.UpgradePoints)
	}
	ship.NextXP = nextXP(ship.Level)
}

func nextXP(level int) int {
	return 8 + level*6
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
