.PHONY: wasm test dev

wasm:
	GOOS=js GOARCH=wasm go build -o static/game.wasm ./cmd/game

test:
	go test ./...

dev:
	ebitdock dev
