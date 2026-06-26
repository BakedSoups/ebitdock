package game

import (
	"image/color"
	"math"
	"math/rand"
)

const (
	ScreenWidth  = 960
	ScreenHeight = 640
)

type Crystal struct {
	X     float64
	Y     float64
	Value int
}

type Upgrades struct {
	Speed    int
	Damage   int
	FireRate int
}

type Ship struct {
	X        float64
	Y        float64
	Angle    float64
	Speed    float64
	Score    int
	Scrap    int
	Alive    bool
	Color    color.RGBA
	Boosting bool
}

type Arena struct {
	Ship     Ship
	Crystals []Crystal
	Tick     int
	Message  string
	Upgrades Upgrades
}

func NewArena() *Arena {
	a := &Arena{
		Ship: Ship{
			X:     ScreenWidth / 2,
			Y:     ScreenHeight / 2,
			Angle: -math.Pi / 2,
			Speed: 2.2,
			Alive: true,
			Color: color.RGBA{R: 75, G: 217, B: 206, A: 255},
		},
		Message: "collect dots, buy upgrades, battle other tabs",
	}
	for i := 0; i < 36; i++ {
		a.SpawnCrystal()
	}
	return a
}

func (a *Arena) SpawnCrystal() {
	a.Crystals = append(a.Crystals, Crystal{
		X:     32 + rand.Float64()*(ScreenWidth-64),
		Y:     32 + rand.Float64()*(ScreenHeight-64),
		Value: 1 + rand.Intn(3),
	})
}

func (a *Arena) Reset() {
	score := a.Ship.Score
	scrap := a.Ship.Scrap
	a.Ship = Ship{
		X:     ScreenWidth / 2,
		Y:     ScreenHeight / 2,
		Angle: -math.Pi / 2,
		Speed: 2.2,
		Alive: true,
		Score: score / 2,
		Scrap: scrap,
		Color: color.RGBA{R: 75, G: 217, B: 206, A: 255},
	}
	a.Message = "respawned with half score"
}
