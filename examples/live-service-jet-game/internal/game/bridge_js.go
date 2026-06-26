//go:build js && wasm

package game

import "syscall/js"

func bridgeReady() {
	global := js.Global()
	bridge := global.Get("OrbitSnake")
	if bridge.IsUndefined() || bridge.IsNull() {
		return
	}
	fn := bridge.Get("ready")
	if fn.Type() == js.TypeFunction {
		fn.Invoke()
	}
}
