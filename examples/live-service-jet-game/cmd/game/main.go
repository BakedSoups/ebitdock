//go:build js && wasm

package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"example.com/orbit-snake/internal/game"
)

func main() {
	ebiten.SetWindowTitle("orbit-snake")
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	if err := ebiten.RunGame(game.New()); err != nil {
		log.Fatal(err)
	}
}
