//go:build js && wasm

package main

import "syscall/js"

func callJSBridge(name string, args ...any) {
	bridge := js.Global().Get("EbitDock")
	if bridge.IsUndefined() || bridge.IsNull() {
		return
	}
	fn := bridge.Get(name)
	if fn.Type() != js.TypeFunction {
		return
	}
	values := make([]any, 0, len(args))
	for _, arg := range args {
		values = append(values, js.ValueOf(arg))
	}
	fn.Invoke(values...)
}
