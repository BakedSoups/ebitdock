package game

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type screenState int

const (
	screenName screenState = iota
	screenPlaying
)

type Game struct {
	state      screenState
	name       string
	local      Blob
	food       []Food
	remotes    []Blob
	cameraX    float64
	cameraY    float64
	status     string
	tick       int
	nextFood   uint64
	rng        *rand.Rand
	lastMouseX float64
	lastMouseY float64
}

func New() *Game {
	g := &Game{
		state: screenName,
		name:  "pilot",
		rng:   rand.New(rand.NewSource(7)),
	}
	g.reset()
	shellReady()
	return g
}

func (g *Game) Update() error {
	g.tick++
	switch g.state {
	case screenName:
		g.updateName()
		return nil
	case screenPlaying:
		g.updatePlaying()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	drawBackground(screen)
	switch g.state {
	case screenName:
		drawNameScreen(screen, g.name, g.status)
	case screenPlaying:
		g.drawWorld(screen)
	}
}

func (g *Game) Layout(int, int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) updateName() {
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(g.name) < 14 && (r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			g.name += string(r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.name) > 0 {
		g.name = g.name[:len(g.name)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.name = strings.TrimSpace(g.name)
		if g.name == "" {
			g.name = "pilot"
		}
		g.local.Name = g.name
		spacetimeJoin(g.name)
		g.state = screenPlaying
	}
	g.status = spacetimeStatus()
}

func (g *Game) updatePlaying() {
	g.status = spacetimeStatus()
	g.applyRemoteSnapshot()

	mouseX, mouseY := ebiten.CursorPosition()
	g.lastMouseX = float64(mouseX)
	g.lastMouseY = float64(mouseY)
	targetX := g.cameraX + float64(mouseX) - ScreenWidth/2
	targetY := g.cameraY + float64(mouseY) - ScreenHeight/2
	spacetimeInput(targetX, targetY)

	dx := targetX - g.local.X
	dy := targetY - g.local.Y
	dist := math.Hypot(dx, dy)
	if dist > 1 {
		speed := math.Max(1.6, 5.4-g.local.Radius*0.035)
		g.local.X += dx / dist * speed
		g.local.Y += dy / dist * speed
		g.local.X = clamp(g.local.X, g.local.Radius, WorldWidth-g.local.Radius)
		g.local.Y = clamp(g.local.Y, g.local.Radius, WorldHeight-g.local.Radius)
	}

	g.eatFood()
	g.eatBots()
	g.updateBots()
	if len(g.food) < 180 {
		for i := 0; i < 4; i++ {
			g.spawnFood()
		}
	}
	g.cameraX = g.local.X
	g.cameraY = g.local.Y
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.reset()
		g.state = screenPlaying
		spacetimeJoin(g.name)
	}
}

func (g *Game) reset() {
	g.local = Blob{
		ID:     "local",
		Name:   g.name,
		X:      WorldWidth / 2,
		Y:      WorldHeight / 2,
		Radius: 22,
		Mass:   22,
		Color:  "#6dd8c7",
		Alive:  true,
	}
	g.food = g.food[:0]
	for len(g.food) < 180 {
		g.spawnFood()
	}
	g.remotes = []Blob{
		{ID: "bot-1", Name: "Byte", X: 380, Y: 360, Radius: 19, Mass: 19, Color: "#ff719a", Alive: true},
		{ID: "bot-2", Name: "Cache", X: 1900, Y: 1220, Radius: 31, Mass: 31, Color: "#ffd166", Alive: true},
		{ID: "bot-3", Name: "Shard", X: 1400, Y: 520, Radius: 25, Mass: 25, Color: "#9d8cff", Alive: true},
	}
	g.cameraX = g.local.X
	g.cameraY = g.local.Y
}

func (g *Game) spawnFood() {
	g.nextFood++
	g.food = append(g.food, Food{
		ID:   g.nextFood,
		X:    32 + g.rng.Float64()*(WorldWidth-64),
		Y:    32 + g.rng.Float64()*(WorldHeight-64),
		Mass: 2 + g.rng.Float64()*3,
	})
}

func (g *Game) eatFood() {
	next := g.food[:0]
	for _, pellet := range g.food {
		if math.Hypot(g.local.X-pellet.X, g.local.Y-pellet.Y) < g.local.Radius {
			g.local.Mass += pellet.Mass * 0.42
			g.local.Radius = radiusForMass(g.local.Mass)
			continue
		}
		next = append(next, pellet)
	}
	g.food = next
}

func (g *Game) eatBots() {
	for i := range g.remotes {
		bot := &g.remotes[i]
		if !bot.Alive {
			continue
		}
		dist := math.Hypot(g.local.X-bot.X, g.local.Y-bot.Y)
		if g.local.Radius > bot.Radius*1.12 && dist < g.local.Radius-bot.Radius*0.25 {
			g.local.Mass += bot.Mass * 0.75
			g.local.Radius = radiusForMass(g.local.Mass)
			bot.Alive = false
		}
		if bot.Radius > g.local.Radius*1.12 && dist < bot.Radius-g.local.Radius*0.25 {
			g.reset()
			g.state = screenName
			g.status = "absorbed; press enter to respawn"
			return
		}
	}
}

func (g *Game) updateBots() {
	for i := range g.remotes {
		bot := &g.remotes[i]
		if !bot.Alive {
			if g.tick%240 == 0 {
				bot.X = 64 + g.rng.Float64()*(WorldWidth-128)
				bot.Y = 64 + g.rng.Float64()*(WorldHeight-128)
				bot.Radius = 18 + g.rng.Float64()*14
				bot.Mass = bot.Radius
				bot.Alive = true
			}
			continue
		}
		angle := math.Sin(float64(g.tick+i*53)*0.013) + float64(i)
		bot.X = clamp(bot.X+math.Cos(angle)*1.2, bot.Radius, WorldWidth-bot.Radius)
		bot.Y = clamp(bot.Y+math.Sin(angle)*1.2, bot.Radius, WorldHeight-bot.Radius)
	}
}

func (g *Game) applyRemoteSnapshot() {
	var snap Snapshot
	if err := json.Unmarshal([]byte(spacetimeSnapshot()), &snap); err != nil || !snap.Connected {
		return
	}
	g.food = snap.Food
	g.remotes = g.remotes[:0]
	for _, player := range snap.Players {
		if player.ID == snap.PlayerID {
			g.local = player
			continue
		}
		g.remotes = append(g.remotes, player)
	}
}

func (g *Game) leaderboard() []Blob {
	players := append([]Blob{g.local}, g.remotes...)
	sort.Slice(players, func(i, j int) bool {
		return players[i].Mass > players[j].Mass
	})
	if len(players) > 5 {
		players = players[:5]
	}
	return players
}

func radiusForMass(mass float64) float64 {
	return math.Sqrt(mass) * 4.7
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

func (g *Game) summary() string {
	return fmt.Sprintf("mass %.0f radius %.0f", g.local.Mass, g.local.Radius)
}
