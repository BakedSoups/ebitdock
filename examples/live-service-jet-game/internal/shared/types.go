package shared

// PlayerProfile is the durable profile returned by the API service.
type PlayerProfile struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color"`
	TotalScrap int      `json:"total_scrap"`
	HighScore  int      `json:"high_score"`
	Upgrades   Upgrades `json:"upgrades"`
}

// Upgrades are intentionally small for the first demo.
type Upgrades struct {
	SpeedLevel  int `json:"speed_level"`
	TurnLevel   int `json:"turn_level"`
	BoostLevel  int `json:"boost_level"`
	ShieldLevel int `json:"shield_level"`
}

// Crystal is a collectable arena item.
type Crystal struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Value  int     `json:"value"`
	Rarity string  `json:"rarity"`
}

// ShipState is the realtime shape exchanged by clients and the realtime server.
type ShipState struct {
	PlayerID string  `json:"player_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Angle    float64 `json:"angle"`
	Speed    float64 `json:"speed"`
	Alive    bool    `json:"alive"`
	Score    int     `json:"score"`
	Level    int     `json:"level"`
	HP       int     `json:"hp"`
	MaxHP    int     `json:"max_hp"`
}

// BulletState is a server-owned projectile in the shared arena.
type BulletState struct {
	ID      string  `json:"id"`
	OwnerID string  `json:"owner_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	VX      float64 `json:"vx"`
	VY      float64 `json:"vy"`
	Damage  int     `json:"damage"`
}
