package build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BakedSoups/ebitdock/internal/config"
	dock "github.com/BakedSoups/ebitdock/internal/docker"
	"github.com/BakedSoups/ebitdock/internal/process"
)

// Result captures the user-visible outcome of a WASM build.
type Result struct {
	Duration time.Duration
	Log      string
	Err      error
}

// WASM builds the configured Go package for the browser target in the
// configured Go Docker image and copies the matching wasm_exec.js from that
// same container toolchain.
func WASM(ctx context.Context, root string, cfg config.Config, status *process.Status) Result {
	start := time.Now()
	status.SetBuild("building", "", nil)
	status.SetBuildLog("")
	status.AppendLog("building wasm")
	if err := dock.RequireDocker(nil); err != nil {
		return finish(status, start, "", err)
	}
	name, args, err := dockerBuildCommand(root, cfg)
	if err != nil {
		return finish(status, start, "", err)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		return finish(status, start, out.String(), fmt.Errorf("wasm docker build failed: %w", err))
	}
	return finish(status, start, out.String(), nil)
}

func dockerBuildCommand(root string, cfg config.Config) (string, []string, error) {
	if cfg.GoImage() == "" {
		return "", nil, errors.New("docker.go_image is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	workdir, target := dockerBuildTarget(root, cfg.Game.Package)
	output := containerPath("/app", cfg.Game.Output)
	wasmExec := containerPath("/app", cfg.WASMExecPath())
	buildFlags := shellWords(cfg.WASMBuildFlags())
	script := strings.Join([]string{
		"mkdir -p " + shellQuote(filepath.Dir(output)) + " " + shellQuote(filepath.Dir(wasmExec)),
		"go build -mod=mod " + buildFlags + " -o " + shellQuote(output) + " " + shellQuote(target),
		"(cp \"$(go env GOROOT)/lib/wasm/wasm_exec.js\" " + shellQuote(wasmExec) + " || cp \"$(go env GOROOT)/misc/wasm/wasm_exec.js\" " + shellQuote(wasmExec) + ")",
	}, " && ")
	args := []string{
		"run", "--rm",
		"-v", absRoot + ":/app",
		"-w", workdir,
		"-e", "GOOS=js",
		"-e", "GOARCH=wasm",
		cfg.GoImage(),
		"sh", "-c", script,
	}
	return "docker", args, nil
}

func dockerBuildTarget(root, pkg string) (workdir, target string) {
	pkgDir := filepath.Join(root, pkg)
	if _, err := os.Stat(filepath.Join(pkgDir, "go.mod")); err == nil {
		return containerPath("/app", pkg), "."
	}
	return "/app", pkg
}

func containerPath(base, path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "./")
	if path == "." {
		return base
	}
	return filepath.ToSlash(filepath.Join(base, path))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func shellWords(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, shellQuote(value))
	}
	return strings.Join(out, " ")
}

// finish updates shared status/logs and returns a compact Result.
func finish(status *process.Status, start time.Time, logText string, err error) Result {
	duration := time.Since(start)
	if logText != "" {
		status.AppendLog(logText)
	}
	status.SetBuildLog(logText)
	if err != nil {
		status.SetBuild("failed", duration.String(), err)
		status.AppendLog(err.Error())
		return Result{Duration: duration, Log: logText, Err: err}
	}
	status.SetBuild("ok", duration.String(), nil)
	status.AppendLog("wasm build completed in " + duration.String())
	return Result{Duration: duration, Log: logText}
}
