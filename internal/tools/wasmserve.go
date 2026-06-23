package tools

import (
	"errors"
	"fmt"
	"os/exec"
)

const WasmserveInstallCommand = "go install github.com/hajimehoshi/wasmserve@latest"

type LookupFunc func(string) (string, error)

func CheckWasmserve(lookup LookupFunc) error {
	if lookup == nil {
		lookup = exec.LookPath
	}
	if _, err := lookup("wasmserve"); err != nil {
		return fmt.Errorf("wasmserve not found\n\nInstall it with:\n  %s", WasmserveInstallCommand)
	}
	return nil
}

func WasmserveCommand(port int, gamePackage string) (string, []string, error) {
	if gamePackage == "" {
		return "", nil, errors.New("game.package is required for wasmserve")
	}
	if port == 0 {
		return "", nil, errors.New("services.web.port is required for wasmserve")
	}
	return "wasmserve", []string{"-http", fmt.Sprintf(":%d", port), gamePackage}, nil
}
