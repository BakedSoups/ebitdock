//go:build js && wasm

package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"example.com/orbit-snake/internal/protocol"
	"example.com/orbit-snake/internal/shared"
)

type NetClient struct {
	mu           sync.RWMutex
	PlayerID     string
	status       string
	socket       js.Value
	reconnecting bool
	ships        []shared.ShipState
	crystals     []shared.Crystal
	bullets      []shared.BulletState
	collected    []string
}

func NewNetClient() *NetClient {
	client := &NetClient{
		PlayerID: randomPlayerID(),
		status:   "connecting",
	}
	client.connect()
	return client
}

func (c *NetClient) SendInput(playerName string, respawn bool, turn int, thrust, shoot bool, x, y, angle float64, upgradePoints, speedLevel, turnLevel, damageLevel, fireRateLevel int) bool {
	if c.socket.IsUndefined() || c.socket.IsNull() || c.socket.Get("readyState").Int() != 1 {
		return false
	}
	c.mu.Lock()
	collected := append([]string(nil), c.collected...)
	c.collected = nil
	c.mu.Unlock()
	msg := protocol.InputMessage{
		Type:          "input",
		PlayerID:      c.PlayerID,
		PlayerName:    playerName,
		Respawn:       respawn,
		Turn:          turn,
		Thrust:        thrust,
		Shoot:         shoot,
		X:             x,
		Y:             y,
		Angle:         angle,
		UpgradePoints: upgradePoints,
		SpeedLevel:    speedLevel,
		TurnLevel:     turnLevel,
		DamageLevel:   damageLevel,
		FireRateLevel: fireRateLevel,
		CollectedIDs:  collected,
	}
	data, err := json.Marshal(msg)
	if err == nil {
		c.socket.Call("send", string(data))
		return true
	}
	return false
}

func (c *NetClient) QueueCrystalCollection(id string) {
	if id == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.collected = append(c.collected, id)
}

func (c *NetClient) Ships() []shared.ShipState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]shared.ShipState(nil), c.ships...)
}

func (c *NetClient) Bullets() []shared.BulletState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]shared.BulletState(nil), c.bullets...)
}

func (c *NetClient) Crystals() []shared.Crystal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]shared.Crystal(nil), c.crystals...)
}

func (c *NetClient) Status() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *NetClient) connect() {
	c.setStatus("connecting")
	wsURL := realtimeURL()
	socket := js.Global().Get("WebSocket").New(wsURL)
	c.socket = socket
	socket.Set("onopen", js.FuncOf(func(js.Value, []js.Value) any {
		c.setStatus("connected")
		return nil
	}))
	socket.Set("onclose", js.FuncOf(func(js.Value, []js.Value) any {
		c.scheduleReconnect("closed")
		return nil
	}))
	socket.Set("onerror", js.FuncOf(func(js.Value, []js.Value) any {
		c.scheduleReconnect("error")
		return nil
	}))
	socket.Set("onmessage", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		var state protocol.StateMessage
		if err := json.Unmarshal([]byte(args[0].Get("data").String()), &state); err == nil {
			c.mu.Lock()
			c.ships = state.Ships
			c.crystals = state.Crystals
			c.bullets = state.Bullets
			c.mu.Unlock()
		}
		return nil
	}))
}

func (c *NetClient) scheduleReconnect(status string) {
	c.mu.Lock()
	if c.reconnecting {
		c.status = status + ", retrying"
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.status = status + ", retrying"
	c.mu.Unlock()

	var timer js.Func
	timer = js.FuncOf(func(js.Value, []js.Value) any {
		timer.Release()
		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
		c.connect()
		return nil
	})
	js.Global().Call("setTimeout", timer, 750)
}

func (c *NetClient) setStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func randomPlayerID() string {
	crypto := js.Global().Get("crypto")
	if !crypto.IsUndefined() && !crypto.IsNull() {
		values := js.Global().Get("Uint32Array").New(2)
		crypto.Call("getRandomValues", values)
		return fmt.Sprintf("p-%08x%08x", values.Index(0).Int(), values.Index(1).Int())
	}
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("p-%06x-%d", rand.Intn(0xffffff), time.Now().UnixNano())
}

func realtimeURL() string {
	location := js.Global().Get("location")
	protocol := "ws:"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	host := location.Get("hostname").String()
	if strings.TrimSpace(host) == "" {
		host = "localhost"
	}
	return protocol + "//" + host + ":3002/ws"
}
