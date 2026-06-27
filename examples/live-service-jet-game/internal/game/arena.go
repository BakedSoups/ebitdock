package game

import (
	"image/color"
	"math"
	"math/rand"

	"example.com/orbit-snake/internal/shared"
)

const (
	ScreenWidth  = 960
	ScreenHeight = 640
)

type Upgrades struct {
	Speed    int
	Turn     int
	Damage   int
	FireRate int
}

type Ship struct {
	X      float64
	Y      float64
	Angle  float64
	Speed  float64
	HP     int
	MaxHP  int
	Score  int
	Scrap  int
	XP     int
	Level  int
	Points int
	Alive  bool
	Color  color.RGBA
}

type Arena struct {
	Ship     Ship
	Crystals []shared.Crystal
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
			HP:    100,
			MaxHP: 100,
			Level: 0,
			Alive: true,
			Color: color.RGBA{R: 75, G: 217, B: 206, A: 255},
		},
		Message: "A/D rotate, W thrust, Space shoot, 1-4 upgrade",
	}
	for i := 0; i < 36; i++ {
		a.SpawnCrystal()
	}
	return a
}

func (a *Arena) SpawnCrystal() {
	a.Crystals = append(a.Crystals, shared.Crystal{
		X:     32 + rand.Float64()*(ScreenWidth-64),
		Y:     32 + rand.Float64()*(ScreenHeight-64),
		Value: 1 + rand.Intn(3),
	})
}

func (a *Arena) Reset() {
	a.Ship = Ship{
		X:     ScreenWidth / 2,
		Y:     ScreenHeight / 2,
		Angle: -math.Pi / 2,
		Speed: 2.2,
		HP:    100,
		MaxHP: 100,
		Alive: true,
		Level: 0,
		Color: color.RGBA{R: 75, G: 217, B: 206, A: 255},
	}
	a.Upgrades = Upgrades{}
	a.Message = "respawned at level 0"
}

func (a *Arena) NextXP() int {
	return 8 + a.Ship.Level*6
}
