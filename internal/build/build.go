package build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"ebitdock/internal/config"
	"ebitdock/internal/process"
)

// Result captures the user-visible outcome of a WASM build.
type Result struct {
	Duration time.Duration
	Log      string
	Err      error
}

// WASM builds the configured Go package for the browser target and copies the
// matching wasm_exec.js from the installed Go toolchain.
func WASM(ctx context.Context, root string, cfg config.Config, status *process.Status) Result {
	start := time.Now()
	status.SetBuild("building", "", nil)
	status.AppendLog("building wasm")

	if _, err := exec.LookPath("go"); err != nil {
		return finish(status, start, "", fmt.Errorf("go executable not found in PATH: %w", err))
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(cfg.Game.Output)), 0o755); err != nil {
		return finish(status, start, "", err)
	}
	if err := CopyWASMExec(root, cfg); err != nil {
		return finish(status, start, "", err)
	}

	buildDir, buildPkg := buildTarget(root, cfg.Game.Package)
	output, err := filepath.Abs(filepath.Join(root, cfg.Game.Output))
	if err != nil {
		return finish(status, start, "", err)
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-mod=mod", "-o", output, buildPkg)
	cmd.Dir = buildDir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		return finish(status, start, out.String(), fmt.Errorf("wasm build failed: %w", err))
	}
	return finish(status, start, out.String(), nil)
}

// buildTarget lets a project keep the game in its own nested Go module. In that
// case we run "go build ." from the nested module and write to an absolute path.
func buildTarget(root, pkg string) (dir, target string) {
	pkgDir := filepath.Join(root, pkg)
	if _, err := os.Stat(filepath.Join(pkgDir, "go.mod")); err == nil {
		return pkgDir, "."
	}
	return root, pkg
}

// CopyWASMExec copies the Go runtime shim that must match the user's installed
// Go version. The location changed across Go releases, so both paths are tried.
func CopyWASMExec(root string, cfg config.Config) error {
	src := filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js")
	if _, err := os.Stat(src); err != nil {
		alt := filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
		if _, altErr := os.Stat(alt); altErr != nil {
			return errors.New("wasm_exec.js not found under GOROOT/lib/wasm or GOROOT/misc/wasm")
		}
		src = alt
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(root, cfg.WASMExecPath())
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// finish updates shared status/logs and returns a compact Result.
func finish(status *process.Status, start time.Time, logText string, err error) Result {
	duration := time.Since(start)
	if logText != "" {
		status.AppendLog(logText)
	}
	if err != nil {
		status.SetBuild("failed", duration.String(), err)
		status.AppendLog(err.Error())
		return Result{Duration: duration, Log: logText, Err: err}
	}
	status.SetBuild("ok", duration.String(), nil)
	status.AppendLog("wasm build completed in " + duration.String())
	return Result{Duration: duration, Log: logText}
}
