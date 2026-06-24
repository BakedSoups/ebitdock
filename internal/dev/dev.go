package dev

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ebitdock/internal/cliui"
	"ebitdock/internal/config"
	"ebitdock/internal/dashboard"
	"ebitdock/internal/process"
	"ebitdock/internal/tools"
	"ebitdock/internal/watch"
)

// Run coordinates the local development session: wasmserve, dashboard,
// optional API process, project logging, and file watching.
func Run(ctx context.Context, root string, cfg config.Config) error {
	status := process.NewStatus(cfg)
	status.SetLogFile(filepath.Join(root, ".ebitdock", "ebitdock.log"))
	status.AppendLog("dev starting")

	if err := tools.CheckWasmserve(nil); err != nil {
		return err
	}
	wasmserveName, wasmserveArgs, err := tools.WasmserveCommand(cfg.WebPort(), cfg.Game.Package)
	if err != nil {
		return err
	}
	wasmserve, err := process.StartArgs(ctx, root, wasmserveName, wasmserveArgs, "wasmserve", status, nil)
	if err != nil {
		return err
	}
	defer wasmserve.Stop()

	var backend *process.Command
	if cfg.APIEnabled() {
		cmd, err := process.Start(ctx, root, cfg.APICommand(), status)
		if err != nil {
			return err
		}
		backend = cmd
		defer backend.Stop()
	}

	// Dashboard is independent from wasmserve. If it fails to bind a port, the
	// error is surfaced through errs and dev exits.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dashboard.RunQuiet(ctx, root, cfg, status); err != nil {
			errs <- err
		}
	}()

	// Watch source and static paths so dashboard/logs reflect local activity.
	// wasmserve owns browser-target compilation during dev.
	changes, watchErrs, err := watch.Changes(ctx, root, cfg.WatchPatterns())
	if err != nil {
		return err
	}
	cliui.DevStatus(os.Stdout, cfg)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case err := <-wasmserve.Done():
			if err != nil {
				return err
			}
			return nil
		case err := <-errs:
			return err
		case err := <-watchErrs:
			if err != nil {
				status.AppendLog("watch error: " + err.Error())
			}
		case path := <-changes:
			if path == "" {
				continue
			}
			if isGeneratedBuildOutput(root, cfg, path) {
				continue
			}
			status.AppendLog("change detected: " + path)
			if isStaticSourceChange(root, cfg, path) {
				status.AppendLog("static file changed")
				continue
			}
			status.AppendLog("source file changed; wasmserve will rebuild on refresh")
		}
	}
}

// isGeneratedBuildOutput prevents rebuild loops when the builder writes
// game.wasm or wasm_exec.js into a watched static directory.
func isGeneratedBuildOutput(root string, cfg config.Config, path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	output, err := filepath.Abs(filepath.Join(root, cfg.Game.Output))
	if err != nil {
		return false
	}
	wasmExec, err := filepath.Abs(filepath.Join(root, cfg.WASMExecPath()))
	if err != nil {
		return false
	}
	return abs == output || abs == wasmExec
}

// isStaticSourceChange reports whether a changed file belongs to the configured
// static root rather than Go source or asset inputs.
func isStaticSourceChange(root string, cfg config.Config, path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return containsPath(root, cfg.StaticRoot(), abs)
}

// containsPath is a path-containment check based on filepath.Rel, avoiding
// brittle string-prefix checks across relative and absolute paths.
func containsPath(root, base, path string) bool {
	baseAbs, err := filepath.Abs(filepath.Join(root, base))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, path)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
