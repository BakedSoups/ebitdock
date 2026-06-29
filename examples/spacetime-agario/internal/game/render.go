package game

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func drawBackground(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 6, G: 13, B: 20, A: 255})
}

func drawNameScreen(screen *ebiten.Image, name, status string) {
	drawBackground(screen)
	panelX := float32(282)
	panelY := float32(210)
	panelW := float32(396)
	panelH := float32(180)
	vector.DrawFilledRect(screen, panelX, panelY, panelW, panelH, color.RGBA{R: 12, G: 24, B: 34, A: 245}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 1, color.RGBA{R: 109, G: 216, B: 199, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, "SPACETIME AGARIO", int(panelX)+118, int(panelY)+28)
	ebitenutil.DebugPrintAt(screen, "NAME", int(panelX)+42, int(panelY)+74)
	vector.DrawFilledRect(screen, panelX+42, panelY+94, panelW-84, 30, color.RGBA{R: 5, G: 11, B: 17, A: 255}, false)
	vector.StrokeRect(screen, panelX+42, panelY+94, panelW-84, 30, 1, color.RGBA{R: 48, G: 76, B: 90, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, name+"_", int(panelX)+54, int(panelY)+103)
	ebitenutil.DebugPrintAt(screen, "ENTER TO JOIN", int(panelX)+140, int(panelY)+142)
	ebitenutil.DebugPrintAt(screen, "net "+status, int(panelX)+42, int(panelY)+160)
}

func (g *Game) drawWorld(screen *ebiten.Image) {
	camX := g.cameraX - ScreenWidth/2
	camY := g.cameraY - ScreenHeight/2
	drawGrid(screen, camX, camY)
	for _, pellet := range g.food {
		x, y := worldToScreen(pellet.X, pellet.Y, camX, camY)
		if x < -10 || x > ScreenWidth+10 || y < -10 || y > ScreenHeight+10 {
			continue
		}
		vector.DrawFilledCircle(screen, float32(x), float32(y), float32(3+pellet.Mass), color.RGBA{R: 111, G: 222, B: 143, A: 255}, false)
	}
	for _, blob := range g.remotes {
		if blob.Alive {
			drawBlob(screen, blob, camX, camY, false)
		}
	}
	drawBlob(screen, g.local, camX, camY, true)
	drawCursorLine(screen, g.lastMouseX, g.lastMouseY)
	drawHUD(screen, g)
}

func drawGrid(screen *ebiten.Image, camX, camY float64) {
	grid := 80.0
	startX := math.Floor(camX/grid) * grid
	startY := math.Floor(camY/grid) * grid
	for x := startX; x < camX+ScreenWidth; x += grid {
		sx := float32(x - camX)
		vector.StrokeLine(screen, sx, 0, sx, ScreenHeight, 1, color.RGBA{R: 20, G: 39, B: 52, A: 255}, false)
	}
	for y := startY; y < camY+ScreenHeight; y += grid {
		sy := float32(y - camY)
		vector.StrokeLine(screen, 0, sy, ScreenWidth, sy, 1, color.RGBA{R: 20, G: 39, B: 52, A: 255}, false)
	}
	left, top := worldToScreen(0, 0, camX, camY)
	right, bottom := worldToScreen(WorldWidth, WorldHeight, camX, camY)
	vector.StrokeRect(screen, float32(left), float32(top), float32(right-left), float32(bottom-top), 3, color.RGBA{R: 80, G: 118, B: 135, A: 255}, false)
}

func drawBlob(screen *ebiten.Image, blob Blob, camX, camY float64, local bool) {
	x, y := worldToScreen(blob.X, blob.Y, camX, camY)
	c := parseHex(blob.Color)
	if local {
		vector.StrokeCircle(screen, float32(x), float32(y), float32(blob.Radius+5), 2, color.RGBA{R: 255, G: 255, B: 255, A: 120}, false)
	}
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(blob.Radius), c, false)
	vector.StrokeCircle(screen, float32(x), float32(y), float32(blob.Radius), 2, color.RGBA{R: 4, G: 8, B: 13, A: 220}, false)
	name := blob.Name
	if name == "" {
		name = blob.ID
	}
	ebitenutil.DebugPrintAt(screen, name, int(x)-len(name)*3, int(y)-4)
}

func drawCursorLine(screen *ebiten.Image, mouseX, mouseY float64) {
	vector.StrokeLine(screen, ScreenWidth/2, ScreenHeight/2, float32(mouseX), float32(mouseY), 1, color.RGBA{R: 109, G: 216, B: 199, A: 100}, false)
}

func drawHUD(screen *ebiten.Image, g *Game) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s  %s  net %s", g.name, g.summary(), g.status), 16, 16)
	ebitenutil.DebugPrintAt(screen, "Mouse to move | R respawn", 16, 34)
	x := ScreenWidth - 214
	y := 16
	vector.DrawFilledRect(screen, float32(x-12), float32(y-8), 198, 120, color.RGBA{R: 8, G: 17, B: 25, A: 220}, false)
	vector.StrokeRect(screen, float32(x-12), float32(y-8), 198, 120, 1, color.RGBA{R: 42, G: 70, B: 84, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, "LEADERS", x, y)
	for i, player := range g.leaderboard() {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d  %-10s %.0f", i+1, player.Name, player.Mass), x, y+20+i*17)
	}
}

func worldToScreen(x, y, camX, camY float64) (float64, float64) {
	return x - camX, y - camY
}

func parseHex(value string) color.RGBA {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return color.RGBA{R: 109, G: 216, B: 199, A: 255}
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return color.RGBA{R: 109, G: 216, B: 199, A: 255}
	}
	return color.RGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}
}
