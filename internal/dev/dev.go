package dev

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ebitdock/internal/build"
	"ebitdock/internal/cliui"
	"ebitdock/internal/config"
	"ebitdock/internal/dashboard"
	"ebitdock/internal/process"
	"ebitdock/internal/serve"
	"ebitdock/internal/watch"
)

func Run(ctx context.Context, root string, cfg config.Config) error {
	status := process.NewStatus(cfg)
	status.SetLogFile(filepath.Join(root, ".ebitdock", "ebitdock.log"))
	status.AppendLog("dev starting")

	var backend *process.Command
	if cfg.APIEnabled() {
		cmd, err := process.Start(ctx, root, cfg.APICommand(), status)
		if err != nil {
			return err
		}
		backend = cmd
		defer backend.Stop()
	}

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := serve.RunQuiet(ctx, root, cfg, status); err != nil {
			errs <- err
		}
	}()
	go func() {
		defer wg.Done()
		if err := dashboard.RunQuiet(ctx, root, cfg, status); err != nil {
			errs <- err
		}
	}()

	result := build.WASM(ctx, root, cfg, status)

	changes, watchErrs, err := watch.Changes(ctx, root, cfg.WatchPatterns())
	if err != nil {
		return err
	}
	cliui.DevStatus(os.Stdout, cfg, result)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
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
			time.Sleep(150 * time.Millisecond)
			if isStaticSourceChange(root, cfg, path) {
				status.AppendLog("static file changed; refresh browser if needed")
				continue
			}
			result := build.WASM(ctx, root, cfg, status)
			cliui.BuildEvent(os.Stdout, result)
		}
	}
}

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

func isStaticSourceChange(root string, cfg config.Config, path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return containsPath(root, cfg.StaticRoot(), abs)
}

func containsPath(root, base, path string) bool {
	baseAbs, err := filepath.Abs(filepath.Join(root, base))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, path)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
