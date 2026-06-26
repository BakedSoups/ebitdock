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

func drawCrystals(screen *ebiten.Image, crystals []shared.Crystal) {
	for _, crystal := range crystals {
		r := float32(4 + crystal.Value*2)
		vector.DrawFilledCircle(screen, float32(crystal.X), float32(crystal.Y), r, color.RGBA{R: 150, G: 105, B: 255, A: 255}, false)
		vector.StrokeCircle(screen, float32(crystal.X), float32(crystal.Y), r+3, 1, color.RGBA{R: 110, G: 245, B: 225, A: 130}, false)
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
	vector.DrawFilledCircle(screen, float32(ship.X-math.Cos(ship.Angle)*17), float32(ship.Y-math.Sin(ship.Angle)*17), 4, color.RGBA{R: 255, G: 201, B: 92, A: 180}, false)
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
		name := ship.PlayerName
		if name == "" {
			name = ship.PlayerID
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s L%d", name, ship.Level), int(ship.X)-22, int(ship.Y)+18)
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

func drawXPBar(screen *ebiten.Image, xp, nextXP, level int) {
	if nextXP <= 0 {
		nextXP = 1
	}
	x := float32(220)
	y := float32(ScreenHeight - 34)
	w := float32(520)
	pct := float32(xp) / float32(nextXP)
	if pct > 1 {
		pct = 1
	}
	vector.DrawFilledRect(screen, x, y, w, 12, color.RGBA{R: 25, G: 30, B: 39, A: 245}, false)
	vector.DrawFilledRect(screen, x, y, w*pct, 12, color.RGBA{R: 96, G: 211, B: 196, A: 255}, false)
	vector.StrokeRect(screen, x, y, w, 12, 1, color.RGBA{R: 128, G: 155, B: 170, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("LEVEL %d   XP %d/%d", level, xp, nextXP), int(x)+174, int(y)-2)
}

func drawUpgradeTree(screen *ebiten.Image, upgrades Upgrades, points int) {
	x := 16
	y := ScreenHeight - 122
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("UPGRADE POINTS %d", points), x, y)
	drawUpgrade(screen, x, y+20, "1 SPEED", upgrades.Speed)
	drawUpgrade(screen, x, y+40, "2 TURN", upgrades.Turn)
	drawUpgrade(screen, x, y+60, "3 DAMAGE", upgrades.Damage)
	drawUpgrade(screen, x, y+80, "4 FIRE", upgrades.FireRate)
}

func drawUpgrade(screen *ebiten.Image, x, y int, label string, level int) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%-8s", label), x, y)
	for i := 0; i < 6; i++ {
		c := color.RGBA{R: 45, G: 55, B: 68, A: 255}
		if i < level {
			c = color.RGBA{R: 255, G: 211, B: 96, A: 255}
		}
		vector.DrawFilledRect(screen, float32(x+84+i*14), float32(y+2), 10, 8, c, false)
	}
}

func drawLoginScreen(screen *ebiten.Image, name, status string) {
	panelX := float32(280)
	panelY := float32(202)
	panelW := float32(400)
	panelH := float32(190)
	vector.DrawFilledRect(screen, panelX, panelY, panelW, panelH, color.RGBA{R: 13, G: 18, B: 25, A: 238}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 1, color.RGBA{R: 93, G: 209, B: 193, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, "ORBIT RAIDERS", int(panelX)+126, int(panelY)+28)
	ebitenutil.DebugPrintAt(screen, "PILOT NAME", int(panelX)+38, int(panelY)+74)
	vector.DrawFilledRect(screen, panelX+38, panelY+96, panelW-76, 28, color.RGBA{R: 7, G: 11, B: 17, A: 255}, false)
	vector.StrokeRect(screen, panelX+38, panelY+96, panelW-76, 28, 1, color.RGBA{R: 64, G: 80, B: 96, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, name+"_", int(panelX)+48, int(panelY)+103)
	ebitenutil.DebugPrintAt(screen, "ENTER TO LAUNCH", int(panelX)+126, int(panelY)+146)
	ebitenutil.DebugPrintAt(screen, "net "+status, int(panelX)+38, int(panelY)+166)
}

func drawEndScreen(screen *ebiten.Image, score, level int) {
	vector.DrawFilledRect(screen, 0, 0, ScreenWidth, ScreenHeight, color.RGBA{R: 4, G: 6, B: 10, A: 175}, false)
	panelX := float32(300)
	panelY := float32(210)
	panelW := float32(360)
	panelH := float32(170)
	vector.DrawFilledRect(screen, panelX, panelY, panelW, panelH, color.RGBA{R: 18, G: 14, B: 20, A: 245}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 1, color.RGBA{R: 255, G: 105, B: 118, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, "SHIP DESTROYED", int(panelX)+104, int(panelY)+32)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("score %d   level %d", score, level), int(panelX)+108, int(panelY)+70)
	ebitenutil.DebugPrintAt(screen, "RESPAWN RESETS TO LEVEL 0", int(panelX)+72, int(panelY)+106)
	ebitenutil.DebugPrintAt(screen, "PRESS R OR ENTER", int(panelX)+110, int(panelY)+132)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
