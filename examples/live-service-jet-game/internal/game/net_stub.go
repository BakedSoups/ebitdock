//go:build !js || !wasm

package game

import "example.com/orbit-snake/internal/shared"

type NetClient struct {
	PlayerID string
}

func NewNetClient() *NetClient {
	return &NetClient{PlayerID: "local"}
}

func (c *NetClient) SendInput(int, bool, bool, bool, float64, float64, float64) {}

func (c *NetClient) Ships() []shared.ShipState {
	return nil
}

func (c *NetClient) Bullets() []shared.BulletState {
	return nil
}

func (c *NetClient) Status() string {
	return "offline"
}
