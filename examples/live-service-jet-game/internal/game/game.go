//go:build js && wasm

package game

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"example.com/orbit-snake/internal/shared"
)

type screenState int

const (
	screenLogin screenState = iota
	screenPlaying
	screenDead
)

type Game struct {
	Arena          *Arena
	Net            *NetClient
	state          screenState
	playerName     string
	lastTurn       int
	thrust         bool
	shoot          bool
	respawnPending bool
	localBullets   []shared.BulletState
	nextBullet     int
	shootCooldown  int
}

func New() *Game {
	bridgeReady()
	return &Game{
		Arena:      NewArena(),
		Net:        NewNetClient(),
		state:      screenLogin,
		playerName: "pilot",
	}
}

func (g *Game) Update() error {
	a := g.Arena
	a.Tick++

	switch g.state {
	case screenLogin:
		g.updateLogin()
		return nil
	case screenDead:
		if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.respawn()
		}
		return nil
	}

	g.syncSelfFromServer()
	if !a.Ship.Alive && !g.respawnPending {
		g.state = screenDead
		return nil
	}

	turn := 0.047 + float64(a.Upgrades.Turn)*0.006
	g.lastTurn = 0
	g.thrust = ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp)
	g.shoot = ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		a.Ship.Angle -= turn
		g.lastTurn = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		a.Ship.Angle += turn
		g.lastTurn = 1
	}

	thrust := 0.025 + float64(a.Upgrades.Speed)*0.004
	if g.thrust {
		a.Ship.Speed += thrust
	}
	a.Ship.Speed *= 0.992
	a.Ship.Speed = clamp(a.Ship.Speed, 0.6, 5.0+float64(a.Upgrades.Speed)*0.45)

	a.Ship.X += math.Cos(a.Ship.Angle) * a.Ship.Speed
	a.Ship.Y += math.Sin(a.Ship.Angle) * a.Ship.Speed
	wrap(&a.Ship.X, ScreenWidth)
	wrap(&a.Ship.Y, ScreenHeight)

	g.collectCrystals()
	g.buyUpgrades()
	g.updateLocalBullets()
	if a.Tick%2 == 0 {
		g.Net.SendInput(g.playerName, g.respawnPending, g.lastTurn, g.thrust, g.shoot, a.Ship.X, a.Ship.Y, a.Ship.Angle, a.Ship.Points, a.Upgrades.Speed, a.Upgrades.Turn, a.Upgrades.Damage, a.Upgrades.FireRate)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	drawBackground(screen)
	if g.state == screenLogin {
		drawLoginScreen(screen, g.playerName, g.Net.Status())
		return
	}
	drawCrystals(screen, g.visibleCrystals())
	drawBullets(screen, g.visibleBullets())
	drawRemoteShips(screen, g.remoteShips())
	drawShip(screen, g.Arena.Ship)
	drawHealthBar(screen, g.Arena.Ship.X-22, g.Arena.Ship.Y-31, g.Arena.Ship.HP, g.Arena.Ship.MaxHP)
	drawXPBar(screen, g.Arena.Ship.XP, g.Arena.NextXP(), g.Arena.Ship.Level)
	drawUpgradeTree(screen, g.Arena.Upgrades, g.Arena.Ship.Points)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("pilot %s  %s  net %s  peers %d", g.playerName, g.selfStats(), g.Net.Status(), len(g.remoteShips())), 16, 14)
	ebitenutil.DebugPrintAt(screen, "A/D rotate | W forward | Space/click shoot | 1 speed 2 turn 3 damage 4 fire", 16, 32)
	ebitenutil.DebugPrintAt(screen, g.Arena.Message, 16, 68)
	if g.state == screenDead {
		drawEndScreen(screen, g.Arena.Ship.Score, g.Arena.Ship.Level)
	}
}

func (g *Game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) collectCrystals() {
	ship := &g.Arena.Ship
	crystals := g.Net.Crystals()
	if len(crystals) == 0 {
		crystals = g.Arena.Crystals
	}
	next := g.Arena.Crystals[:0]
	for _, crystal := range crystals {
		if distance(ship.X, ship.Y, crystal.X, crystal.Y) < 16 {
			if crystal.ID != "" {
				g.Net.QueueCrystalCollection(crystal.ID)
				g.Arena.Message = fmt.Sprintf("+%d XP queued", crystal.Value)
			} else {
				ship.Score += crystal.Value
				g.addXP(crystal.Value)
				g.Arena.Message = fmt.Sprintf("+%d XP", crystal.Value)
				g.Arena.SpawnCrystal()
			}
			continue
		}
		next = append(next, crystal)
	}
	if len(g.Net.Crystals()) == 0 {
		g.Arena.Crystals = next
	}
}

func (g *Game) buyUpgrades() {
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.buyUpgrade("speed", &g.Arena.Upgrades.Speed)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.buyUpgrade("turn", &g.Arena.Upgrades.Turn)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.buyUpgrade("damage", &g.Arena.Upgrades.Damage)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.buyUpgrade("fire", &g.Arena.Upgrades.FireRate)
	}
}

func (g *Game) buyUpgrade(name string, level *int) {
	if g.Arena.Ship.Points <= 0 {
		g.Arena.Message = fmt.Sprintf("%s upgrade needs a level point", name)
		return
	}
	g.Arena.Ship.Points--
	*level = *level + 1
	g.Arena.Message = fmt.Sprintf("%s upgraded to %d", name, *level)
}

func (g *Game) updateLocalBullets() {
	if g.shootCooldown > 0 {
		g.shootCooldown--
	}
	if g.shoot && g.shootCooldown == 0 {
		g.nextBullet++
		speed := 7.4 + float64(g.Arena.Upgrades.FireRate)*0.35
		g.localBullets = append(g.localBullets, shared.BulletState{
			ID:      fmt.Sprintf("local-%d", g.nextBullet),
			OwnerID: g.Net.PlayerID,
			X:       g.Arena.Ship.X + math.Cos(g.Arena.Ship.Angle)*20,
			Y:       g.Arena.Ship.Y + math.Sin(g.Arena.Ship.Angle)*20,
			VX:      math.Cos(g.Arena.Ship.Angle) * speed,
			VY:      math.Sin(g.Arena.Ship.Angle) * speed,
			Damage:  1,
		})
		g.shootCooldown = max(6, 20-g.Arena.Upgrades.FireRate*2)
	}
	next := g.localBullets[:0]
	for _, bullet := range g.localBullets {
		bullet.X += bullet.VX
		bullet.Y += bullet.VY
		if bullet.X < -30 || bullet.X > ScreenWidth+30 || bullet.Y < -30 || bullet.Y > ScreenHeight+30 {
			continue
		}
		next = append(next, bullet)
	}
	g.localBullets = next
}

func (g *Game) updateLogin() {
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(g.playerName) < 14 && (r == '-' || r == '_' || r == ' ' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			g.playerName += string(r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.playerName) > 0 {
		g.playerName = g.playerName[:len(g.playerName)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.playerName = strings.TrimSpace(g.playerName)
		if g.playerName == "" {
			g.playerName = "pilot"
		}
		g.respawn()
	}
}

func (g *Game) respawn() {
	g.Arena.Reset()
	g.localBullets = nil
	g.shootCooldown = 0
	g.respawnPending = true
	g.state = screenPlaying
	g.Net.SendInput(g.playerName, true, 0, false, false, g.Arena.Ship.X, g.Arena.Ship.Y, g.Arena.Ship.Angle, 0, 0, 0, 0, 0)
}

func (g *Game) addXP(amount int) {
	g.Arena.Ship.XP += amount
	for g.Arena.Ship.XP >= g.Arena.NextXP() {
		g.Arena.Ship.XP -= g.Arena.NextXP()
		g.Arena.Ship.Level++
		g.Arena.Ship.Points++
		g.Arena.Message = fmt.Sprintf("level %d reached: choose an upgrade", g.Arena.Ship.Level)
	}
}

func (g *Game) syncSelfFromServer() {
	for _, ship := range g.Net.Ships() {
		if ship.PlayerID != g.Net.PlayerID {
			continue
		}
		g.Arena.Ship.Alive = ship.Alive
		g.Arena.Ship.Score = ship.Score
		g.Arena.Ship.Level = ship.Level
		g.Arena.Ship.XP = ship.XP
		g.Arena.Ship.Points = ship.UpgradePoints
		g.Arena.Ship.HP = ship.HP
		g.Arena.Ship.MaxHP = ship.MaxHP
		if ship.Alive {
			g.respawnPending = false
		}
		return
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

func (g *Game) visibleCrystals() []shared.Crystal {
	crystals := g.Net.Crystals()
	if len(crystals) > 0 {
		return crystals
	}
	return g.Arena.Crystals
}

func (g *Game) visibleBullets() []shared.BulletState {
	server := g.Net.Bullets()
	if len(server) == 0 {
		return g.localBullets
	}
	out := append([]shared.BulletState(nil), server...)
	out = append(out, g.localBullets...)
	return out
}

func (g *Game) selfStats() string {
	for _, ship := range g.Net.Ships() {
		if ship.PlayerID == g.Net.PlayerID {
			return fmt.Sprintf("hp %d/%d lvl %d kills %d", ship.HP, ship.MaxHP, ship.Level, ship.Score)
		}
	}
	return fmt.Sprintf("hp %d/%d lvl %d kills %d", g.Arena.Ship.HP, g.Arena.Ship.MaxHP, g.Arena.Ship.Level, g.Arena.Ship.Score)
}
