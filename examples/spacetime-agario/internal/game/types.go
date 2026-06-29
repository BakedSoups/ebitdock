package game

const (
	ScreenWidth  = 960
	ScreenHeight = 640
	WorldWidth   = 2400
	WorldHeight  = 1800
)

type Blob struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Mass   float64 `json:"mass"`
	Color  string  `json:"color"`
	Alive  bool    `json:"alive"`
}

type Food struct {
	ID   uint64  `json:"id"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Mass float64 `json:"mass"`
}

type Snapshot struct {
	Connected bool   `json:"connected"`
	PlayerID  string `json:"playerId"`
	Players   []Blob `json:"players"`
	Food      []Food `json:"food"`
}
