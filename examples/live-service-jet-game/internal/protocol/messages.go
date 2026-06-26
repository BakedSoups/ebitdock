package protocol

import "example.com/orbit-snake/internal/shared"

// InputMessage is sent by a browser client to the realtime service.
type InputMessage struct {
	Type          string   `json:"type"`
	PlayerID      string   `json:"player_id"`
	PlayerName    string   `json:"player_name,omitempty"`
	Respawn       bool     `json:"respawn,omitempty"`
	Turn          int      `json:"turn"`
	Thrust        bool     `json:"thrust"`
	Shoot         bool     `json:"shoot"`
	X             float64  `json:"x"`
	Y             float64  `json:"y"`
	Angle         float64  `json:"angle"`
	UpgradePoints int      `json:"upgrade_points"`
	SpeedLevel    int      `json:"speed_level"`
	TurnLevel     int      `json:"turn_level"`
	DamageLevel   int      `json:"damage_level"`
	FireRateLevel int      `json:"fire_rate_level"`
	CollectedIDs  []string `json:"collected_ids,omitempty"`
}

// StateMessage is broadcast by the realtime service.
type StateMessage struct {
	Type     string               `json:"type"`
	Ships    []shared.ShipState   `json:"ships"`
	Crystals []shared.Crystal     `json:"crystals"`
	Bullets  []shared.BulletState `json:"bullets"`
	Events   []Event              `json:"events"`
}

// Event is a compact arena event for HUD/log display.
type Event struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id,omitempty"`
	Message  string `json:"message,omitempty"`
}
