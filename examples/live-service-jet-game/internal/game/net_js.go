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
	mu       sync.RWMutex
	PlayerID string
	status   string
	socket   js.Value
	ships    []shared.ShipState
}

func NewNetClient() *NetClient {
	client := &NetClient{
		PlayerID: tabPlayerID(),
		status:   "connecting",
	}
	client.connect()
	return client
}

func (c *NetClient) SendInput(turn int, thrust, boost bool) {
	if c.socket.IsUndefined() || c.socket.IsNull() || c.socket.Get("readyState").Int() != 1 {
		return
	}
	msg := protocol.InputMessage{
		Type:     "input",
		PlayerID: c.PlayerID,
		Turn:     turn,
		Thrust:   thrust,
		Boost:    boost,
	}
	data, err := json.Marshal(msg)
	if err == nil {
		c.socket.Call("send", string(data))
	}
}

func (c *NetClient) Ships() []shared.ShipState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]shared.ShipState(nil), c.ships...)
}

func (c *NetClient) Status() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *NetClient) connect() {
	wsURL := realtimeURL()
	socket := js.Global().Get("WebSocket").New(wsURL)
	c.socket = socket
	socket.Set("onopen", js.FuncOf(func(js.Value, []js.Value) any {
		c.setStatus("connected")
		c.SendInput(0, false, false)
		return nil
	}))
	socket.Set("onclose", js.FuncOf(func(js.Value, []js.Value) any {
		c.setStatus("closed")
		return nil
	}))
	socket.Set("onerror", js.FuncOf(func(js.Value, []js.Value) any {
		c.setStatus("error")
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
			c.mu.Unlock()
		}
		return nil
	}))
}

func (c *NetClient) setStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func tabPlayerID() string {
	storage := js.Global().Get("sessionStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return randomPlayerID()
	}
	existing := storage.Call("getItem", "orbitSnakePlayerID")
	if !existing.IsNull() && existing.String() != "" {
		return existing.String()
	}
	id := randomPlayerID()
	storage.Call("setItem", "orbitSnakePlayerID", id)
	return id
}

func randomPlayerID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("p-%06x", rand.Intn(0xffffff))
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
