package tools

import (
	"errors"
	"fmt"
	"os/exec"
)

// WasmserveInstallCommand is included verbatim in missing-tool errors.
const WasmserveInstallCommand = "go install github.com/hajimehoshi/wasmserve@latest"

// LookupFunc lets tests replace exec.LookPath without touching PATH.
type LookupFunc func(string) (string, error)

// CheckWasmserve verifies that the external wasmserve command is installed.
func CheckWasmserve(lookup LookupFunc) error {
	if lookup == nil {
		lookup = exec.LookPath
	}
	if _, err := lookup("wasmserve"); err != nil {
		return fmt.Errorf("wasmserve not found\n\nInstall it with:\n  %s", WasmserveInstallCommand)
	}
	return nil
}

// WasmserveCommand builds the command line ebitdock will use once dev mode is
// wired to wasmserve.
func WasmserveCommand(port int, gamePackage string) (string, []string, error) {
	if gamePackage == "" {
		return "", nil, errors.New("game.package is required for wasmserve")
	}
	if port == 0 {
		return "", nil, errors.New("services.web.port is required for wasmserve")
	}
	return "wasmserve", []string{"-http", fmt.Sprintf(":%d", port), gamePackage}, nil
}
