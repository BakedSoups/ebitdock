package protocol

import "example.com/orbit-snake/internal/shared"

// InputMessage is sent by a browser client to the realtime service.
type InputMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id"`
	Turn     int    `json:"turn"`
	Thrust   bool   `json:"thrust"`
	Boost    bool   `json:"boost"`
}

// StateMessage is broadcast by the realtime service.
type StateMessage struct {
	Type     string             `json:"type"`
	Ships    []shared.ShipState `json:"ships"`
	Crystals []shared.Crystal   `json:"crystals"`
	Events   []Event            `json:"events"`
}

// Event is a compact arena event for HUD/log display.
type Event struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id,omitempty"`
	Message  string `json:"message,omitempty"`
}
