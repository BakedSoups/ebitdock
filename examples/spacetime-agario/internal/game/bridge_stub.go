//go:build !js || !wasm

package game

func shellReady() {}

func spacetimeStatus() string {
	return "offline"
}

func spacetimeJoin(string) {}

func spacetimeInput(float64, float64) {}

func spacetimeSnapshot() string {
	return "{}"
}
