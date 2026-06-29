//go:build js && wasm

package game

import "syscall/js"

func shellReady() {
	bridge := js.Global().Get("SpacetimeAgarioShell")
	if bridge.Truthy() {
		bridge.Call("ready")
	}
}

func spacetimeStatus() string {
	bridge := js.Global().Get("SpacetimeAgario")
	if !bridge.Truthy() {
		return "missing bridge"
	}
	return bridge.Call("status").String()
}

func spacetimeJoin(name string) {
	bridge := js.Global().Get("SpacetimeAgario")
	if bridge.Truthy() {
		bridge.Call("join", name)
	}
}

func spacetimeInput(x, y float64) {
	bridge := js.Global().Get("SpacetimeAgario")
	if bridge.Truthy() {
		bridge.Call("input", x, y)
	}
}

func spacetimeSnapshot() string {
	bridge := js.Global().Get("SpacetimeAgario")
	if !bridge.Truthy() {
		return "{}"
	}
	return bridge.Call("snapshot").String()
}
