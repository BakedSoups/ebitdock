//go:build js && wasm

package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	Arena *Arena
}

func New() *Game {
	bridgeReady()
	return &Game{Arena: NewArena()}
}

func (g *Game) Update() error {
	a := g.Arena
	a.Tick++
	if !a.Ship.Alive {
		if ebiten.IsKeyPressed(ebiten.KeyR) {
			a.Reset()
		}
		return nil
	}

	turn := 0.045
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		a.Ship.Angle -= turn
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		a.Ship.Angle += turn
	}

	thrust := 0.02
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		a.Ship.Speed += thrust
	}
	a.Ship.Boosting = ebiten.IsKeyPressed(ebiten.KeySpace) && a.Ship.Scrap > 0
	if a.Ship.Boosting {
		a.Ship.Speed += 0.04
		if a.Tick%12 == 0 {
			a.Ship.Scrap--
		}
	}
	a.Ship.Speed *= 0.992
	a.Ship.Speed = clamp(a.Ship.Speed, 1.8, 5.0)

	a.Ship.X += math.Cos(a.Ship.Angle) * a.Ship.Speed
	a.Ship.Y += math.Sin(a.Ship.Angle) * a.Ship.Speed
	wrap(&a.Ship.X, ScreenWidth)
	wrap(&a.Ship.Y, ScreenHeight)

	if a.Tick%3 == 0 {
		a.Ship.Trail = append(a.Ship.Trail, TrailPoint{X: a.Ship.X, Y: a.Ship.Y})
		limit := 42 + a.Ship.Score/2
		if len(a.Ship.Trail) > limit {
			a.Ship.Trail = a.Ship.Trail[len(a.Ship.Trail)-limit:]
		}
	}

	g.collectCrystals()
	g.checkTrailCollision()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	drawBackground(screen)
	drawCrystals(screen, g.Arena.Crystals)
	drawTrail(screen, g.Arena.Ship)
	drawShip(screen, g.Arena.Ship)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("score %d  scrap %d  trail %d", g.Arena.Ship.Score, g.Arena.Ship.Scrap, len(g.Arena.Ship.Trail)), 16, 14)
	ebitenutil.DebugPrintAt(screen, "WASD/arrow turn + thrust | space boost | R respawn", 16, 32)
	ebitenutil.DebugPrintAt(screen, g.Arena.Message, 16, 50)
}

func (g *Game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) collectCrystals() {
	ship := &g.Arena.Ship
	next := g.Arena.Crystals[:0]
	for _, crystal := range g.Arena.Crystals {
		if distance(ship.X, ship.Y, crystal.X, crystal.Y) < 16 {
			ship.Score += crystal.Value
			ship.Scrap += crystal.Value
			g.Arena.Message = fmt.Sprintf("+%d crystal", crystal.Value)
			g.Arena.SpawnCrystal()
			continue
		}
		next = append(next, crystal)
	}
	g.Arena.Crystals = next
}

func (g *Game) checkTrailCollision() {
	ship := &g.Arena.Ship
	if len(ship.Trail) < 24 {
		return
	}
	for _, point := range ship.Trail[:len(ship.Trail)-18] {
		if distance(ship.X, ship.Y, point.X, point.Y) < 9 {
			ship.Alive = false
			g.Arena.Message = "trail collision: press R to respawn"
			return
		}
	}
}

func wrap(v *float64, max float64) {
	if *v < 0 {
		*v += max
	}
	if *v > max {
		*v -= max
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func distance(ax, ay, bx, by float64) float64 {
	return math.Hypot(ax-bx, ay-by)
}
