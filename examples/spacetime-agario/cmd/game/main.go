package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"example.com/spacetime-agario/internal/game"
)

func main() {
	ebiten.SetWindowTitle("spacetime-agario")
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	if err := ebiten.RunGame(game.New()); err != nil {
		log.Fatal(err)
	}
}
