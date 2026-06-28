package tools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WasmservePackage is the Go package installed by ebitdock install tools.
const WasmservePackage = "github.com/hajimehoshi/wasmserve@latest"

// WasmserveInstallCommand is included verbatim in missing-tool errors.
const WasmserveInstallCommand = "go install " + WasmservePackage

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

// WasmserveCommand builds the command line for Ebitengine browser development.
func WasmserveCommand(port int, gamePackage string) (string, []string, error) {
	if gamePackage == "" {
		return "", nil, errors.New("game.package is required for wasmserve")
	}
	if port == 0 {
		return "", nil, errors.New("services.web.port is required for wasmserve")
	}
	return "wasmserve", []string{"-http", fmt.Sprintf(":%d", port), gamePackage}, nil
}

// NotifyWasmserve tells wasmserve to wake any browser pages waiting on /_wait.
func NotifyWasmserve(ctx context.Context, port int) error {
	if port == 0 {
		return errors.New("services.web.port is required for wasmserve notify")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/_notify", port), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("wasmserve notify returned %s", resp.Status)
	}
	return nil
}

// BrowserShellHints reports whether a user-owned browser shell appears to use
// wasmserve's dev-only main.wasm and /_wait endpoints.
func BrowserShellHints(root, staticRoot string) []string {
	indexPath := filepath.Join(root, staticRoot, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}
	html := string(data)
	var hints []string
	if strings.Contains(html, "game.wasm") && !strings.Contains(html, "main.wasm") {
		hints = append(hints, "static/index.html loads game.wasm; wasmserve rebuilds only main.wasm during dev")
	}
	if !strings.Contains(html, "_wait") {
		hints = append(hints, "static/index.html does not wait on /_wait; /_notify will not auto-reload the browser")
	}
	return hints
}
