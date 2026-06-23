//go:build !js || !wasm

package main

func callJSBridge(name string, args ...any) {}
