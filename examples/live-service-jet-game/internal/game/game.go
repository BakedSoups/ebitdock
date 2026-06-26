//go:build js && wasm

package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"example.com/orbit-snake/internal/shared"
)

type Game struct {
	Arena    *Arena
	Net      *NetClient
	lastTurn int
	thrust   bool
	boost    bool
	shoot    bool
}

func New() *Game {
	bridgeReady()
	return &Game{
		Arena: NewArena(),
		Net:   NewNetClient(),
	}
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
	g.lastTurn = 0
	g.thrust = ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp)
	g.boost = (ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)) && a.Ship.Scrap > 0
	g.shoot = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsKeyPressed(ebiten.KeySpace)
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		a.Ship.Angle -= turn
		g.lastTurn = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		a.Ship.Angle += turn
		g.lastTurn = 1
	}
	if mx, my := ebiten.CursorPosition(); mx != 0 || my != 0 {
		a.Ship.Angle = math.Atan2(float64(my)-a.Ship.Y, float64(mx)-a.Ship.X)
	}

	thrust := 0.02
	if g.thrust {
		a.Ship.Speed += thrust
	}
	a.Ship.Boosting = g.boost
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

	g.collectCrystals()
	g.buyUpgrades()
	if a.Tick%2 == 0 {
		g.Net.SendInput(g.lastTurn, g.thrust, g.boost, g.shoot, a.Ship.X, a.Ship.Y, a.Ship.Angle, a.Upgrades.Speed, a.Upgrades.Damage, a.Upgrades.FireRate)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	drawBackground(screen)
	drawCrystals(screen, g.Arena.Crystals)
	drawBullets(screen, g.Net.Bullets())
	drawRemoteShips(screen, g.remoteShips())
	drawShip(screen, g.Arena.Ship)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("scrap %d  upgrades speed %d damage %d fire %d  server %s", g.Arena.Ship.Scrap, g.Arena.Upgrades.Speed, g.Arena.Upgrades.Damage, g.Arena.Upgrades.FireRate, g.selfStats()), 16, 14)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("player %s  net %s  peers %d", g.Net.PlayerID, g.Net.Status(), len(g.remoteShips())), 16, 32)
	ebitenutil.DebugPrintAt(screen, "WASD/arrow move | mouse aim | click/space shoot | shift boost | dots buy 1 speed 2 damage 3 fire", 16, 50)
	ebitenutil.DebugPrintAt(screen, g.Arena.Message, 16, 68)
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
			g.Arena.Message = fmt.Sprintf("+%d scrap dot", crystal.Value)
			g.Arena.SpawnCrystal()
			continue
		}
		next = append(next, crystal)
	}
	g.Arena.Crystals = next
}

func (g *Game) buyUpgrades() {
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.buyUpgrade("speed", &g.Arena.Upgrades.Speed)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.buyUpgrade("damage", &g.Arena.Upgrades.Damage)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.buyUpgrade("fire", &g.Arena.Upgrades.FireRate)
	}
}

func (g *Game) buyUpgrade(name string, level *int) {
	cost := 4 + (*level * 3)
	if g.Arena.Ship.Scrap < cost {
		g.Arena.Message = fmt.Sprintf("%s upgrade needs %d scrap", name, cost)
		return
	}
	g.Arena.Ship.Scrap -= cost
	*level = *level + 1
	g.Arena.Message = fmt.Sprintf("%s upgraded to %d", name, *level)
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

func (g *Game) remoteShips() []shared.ShipState {
	ships := g.Net.Ships()
	out := ships[:0]
	for _, ship := range ships {
		if ship.PlayerID == g.Net.PlayerID {
			continue
		}
		out = append(out, ship)
	}
	return out
}

func (g *Game) selfStats() string {
	for _, ship := range g.Net.Ships() {
		if ship.PlayerID == g.Net.PlayerID {
			return fmt.Sprintf("hp %d/%d lvl %d kills %d", ship.HP, ship.MaxHP, ship.Level, ship.Score)
		}
	}
	return "waiting"
}
