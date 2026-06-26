//go:build js && wasm

package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"example.com/orbit-snake/internal/shared"
)

func drawBackground(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 7, G: 9, B: 13, A: 255})
	for x := 0; x < ScreenWidth; x += 48 {
		for y := 0; y < ScreenHeight; y += 48 {
			if (x+y)%96 == 0 {
				vector.DrawFilledCircle(screen, float32(x+11), float32(y+17), 1.2, color.RGBA{R: 48, G: 69, B: 82, A: 255}, false)
			}
		}
	}
}

func drawCrystals(screen *ebiten.Image, crystals []Crystal) {
	for _, crystal := range crystals {
		r := float32(4 + crystal.Value*2)
		vector.DrawFilledCircle(screen, float32(crystal.X), float32(crystal.Y), r, color.RGBA{R: 150, G: 105, B: 255, A: 255}, false)
		vector.StrokeCircle(screen, float32(crystal.X), float32(crystal.Y), r+3, 1, color.RGBA{R: 110, G: 245, B: 225, A: 130}, false)
	}
}

func drawTrail(screen *ebiten.Image, ship Ship) {
	for i, point := range ship.Trail {
		alpha := uint8(40 + i*180/max(1, len(ship.Trail)))
		vector.DrawFilledCircle(screen, float32(point.X), float32(point.Y), 5, color.RGBA{R: ship.Color.R, G: ship.Color.G, B: ship.Color.B, A: alpha}, false)
	}
}

func drawShip(screen *ebiten.Image, ship Ship) {
	if !ship.Alive {
		vector.StrokeCircle(screen, float32(ship.X), float32(ship.Y), 20, 2, color.RGBA{R: 255, G: 90, B: 100, A: 255}, false)
		return
	}
	noseX := ship.X + math.Cos(ship.Angle)*18
	noseY := ship.Y + math.Sin(ship.Angle)*18
	leftX := ship.X + math.Cos(ship.Angle+2.45)*13
	leftY := ship.Y + math.Sin(ship.Angle+2.45)*13
	rightX := ship.X + math.Cos(ship.Angle-2.45)*13
	rightY := ship.Y + math.Sin(ship.Angle-2.45)*13

	vector.DrawFilledCircle(screen, float32(ship.X), float32(ship.Y), 10, color.RGBA{R: 13, G: 20, B: 28, A: 255}, false)
	vector.StrokeLine(screen, float32(noseX), float32(noseY), float32(leftX), float32(leftY), 3, ship.Color, false)
	vector.StrokeLine(screen, float32(leftX), float32(leftY), float32(rightX), float32(rightY), 3, ship.Color, false)
	vector.StrokeLine(screen, float32(rightX), float32(rightY), float32(noseX), float32(noseY), 3, ship.Color, false)
	if ship.Boosting {
		vector.DrawFilledCircle(screen, float32(ship.X-math.Cos(ship.Angle)*17), float32(ship.Y-math.Sin(ship.Angle)*17), 6, color.RGBA{R: 255, G: 201, B: 92, A: 220}, false)
	}
}

func drawRemoteShips(screen *ebiten.Image, ships []shared.ShipState) {
	for _, ship := range ships {
		noseX := ship.X + math.Cos(ship.Angle)*15
		noseY := ship.Y + math.Sin(ship.Angle)*15
		leftX := ship.X + math.Cos(ship.Angle+2.45)*11
		leftY := ship.Y + math.Sin(ship.Angle+2.45)*11
		rightX := ship.X + math.Cos(ship.Angle-2.45)*11
		rightY := ship.Y + math.Sin(ship.Angle-2.45)*11
		body := color.RGBA{R: 255, G: 124, B: 178, A: 255}
		vector.DrawFilledCircle(screen, float32(ship.X), float32(ship.Y), 7, color.RGBA{R: 30, G: 12, B: 24, A: 220}, false)
		vector.StrokeLine(screen, float32(noseX), float32(noseY), float32(leftX), float32(leftY), 2, body, false)
		vector.StrokeLine(screen, float32(leftX), float32(leftY), float32(rightX), float32(rightY), 2, body, false)
		vector.StrokeLine(screen, float32(rightX), float32(rightY), float32(noseX), float32(noseY), 2, body, false)
		drawHealthBar(screen, ship.X-18, ship.Y-25, ship.HP, ship.MaxHP)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("L%d K%d", ship.Level, ship.Score), int(ship.X)-16, int(ship.Y)+18)
	}
}

func drawBullets(screen *ebiten.Image, bullets []shared.BulletState) {
	for _, bullet := range bullets {
		vector.DrawFilledCircle(screen, float32(bullet.X), float32(bullet.Y), 4, color.RGBA{R: 255, G: 224, B: 112, A: 255}, false)
		vector.StrokeCircle(screen, float32(bullet.X), float32(bullet.Y), 7, 1, color.RGBA{R: 255, G: 126, B: 72, A: 180}, false)
	}
}

func drawHealthBar(screen *ebiten.Image, x, y float64, hp, maxHP int) {
	if maxHP <= 0 {
		maxHP = 1
	}
	width := float32(36)
	pct := float32(hp) / float32(maxHP)
	if pct < 0 {
		pct = 0
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), width, 4, color.RGBA{R: 48, G: 24, B: 30, A: 255}, false)
	vector.DrawFilledRect(screen, float32(x), float32(y), width*pct, 4, color.RGBA{R: 95, G: 231, B: 146, A: 255}, false)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
