package dev

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BakedSoups/ebitdock/internal/build"
	"github.com/BakedSoups/ebitdock/internal/checks"
	"github.com/BakedSoups/ebitdock/internal/cliui"
	"github.com/BakedSoups/ebitdock/internal/config"
	"github.com/BakedSoups/ebitdock/internal/dashboard"
	dock "github.com/BakedSoups/ebitdock/internal/docker"
	"github.com/BakedSoups/ebitdock/internal/process"
	"github.com/BakedSoups/ebitdock/internal/watch"
)

// Run coordinates the Docker-backed development session: Compose services,
// dashboard, project logging, and file watching.
func Run(ctx context.Context, root string, cfg config.Config) error {
	status := process.NewStatus(cfg)
	status.SetLogFile(filepath.Join(root, ".ebitdock", "ebitdock.log"))
	status.AppendLog("dev starting")
	return runDocker(ctx, root, cfg, status)
}

func runDocker(ctx context.Context, root string, cfg config.Config, status *process.Status) error {
	if err := dock.RequireDocker(nil); err != nil {
		return err
	}
	composePath, err := dock.WriteCompose(root, cfg)
	if err != nil {
		return err
	}
	status.AppendLog("wrote compose file: " + composePath)

	if err := checkedBuild(ctx, root, cfg, status, os.Stdout); err != nil {
		return err
	}

	name, args, err := dock.ComposeCommand(cfg.ComposeFile(), "up", "--build", "--remove-orphans")
	if err != nil {
		return err
	}
	var setBackend func(string)
	if cfg.APIEnabled() {
		setBackend = status.SetServer
	}
	stack, err := process.StartArgs(ctx, root, name, args, "docker", status, setBackend)
	if err != nil {
		return err
	}
	status.SetServices("running")
	defer func() {
		stack.Stop()
		status.SetServices("stopped")
		dockerComposeDown(root, cfg, status)
	}()

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dashboard.RunQuiet(ctx, root, cfg, status); err != nil {
			errs <- err
		}
	}()

	changes, watchErrs, err := watch.Changes(ctx, root, cfg.WatchPatterns())
	if err != nil {
		return err
	}
	cliui.DevStatus(os.Stdout, cfg, "docker")

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case err := <-stack.Done():
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
			if path == "" || isGeneratedBuildOutput(root, cfg, path) {
				continue
			}
			status.SetLastChange(path)
			status.AppendLog("change detected: " + path)
			fmt.Fprintln(os.Stdout, "change\t"+path)
			if isStaticSourceChange(root, cfg, path) {
				status.AppendLog("static file changed")
				continue
			}
			if err := checkedBuild(ctx, root, cfg, status, os.Stdout); err != nil {
				continue
			}
		}
	}
}

func dockerComposeDown(root string, cfg config.Config, status *process.Status) {
	name, args, err := dock.ComposeCommand(cfg.ComposeFile(), "down", "--remove-orphans")
	if err != nil {
		status.AppendLog("docker down skipped: " + err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		status.AppendLog(string(out))
	}
	if err != nil {
		status.AppendLog("docker down failed: " + err.Error())
	}
}

func printBuildResult(out io.Writer, result build.Result) {
	cliui.Result(out, "build", result.Duration, result.Err)
	if result.Err != nil {
		printStepLog(out, result.Log)
	}
}

func checkedBuild(ctx context.Context, root string, cfg config.Config, status *process.Status, out io.Writer) error {
	if cfg.BeforeRebuildCheckEnabled() {
		activity := cliui.StartActivity(out, "check", "running", cfg.BeforeRebuildCheckCommand())
		result := checks.Run(ctx, root, cfg.BeforeRebuildCheckCommand(), status)
		activity.Stop()
		printCheckResult(out, result)
		if result.Err != nil {
			return result.Err
		}
	}
	status.AppendLog("rebuilding wasm")
	buildDetail := fmt.Sprintf("GOOS=js GOARCH=wasm go build %s -o %s %s", strings.Join(cfg.WASMBuildFlags(), " "), cfg.Game.Output, cfg.Game.Package)
	activity := cliui.StartActivity(out, "build", "running", strings.Join(strings.Fields(buildDetail), " "))
	result := build.WASM(ctx, root, cfg, status)
	activity.Stop()
	printBuildResult(out, result)
	return result.Err
}

func printCheckResult(out io.Writer, result checks.Result) {
	cliui.Result(out, "check", result.Duration, result.Err)
	if result.Err != nil {
		printStepLog(out, result.Log)
	}
}

func printStepLog(out io.Writer, logText string) {
	logText = strings.TrimSpace(logText)
	if logText == "" {
		return
	}
	fmt.Fprintln(out, logText)
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
	return abs == output || abs == wasmExec || strings.HasPrefix(abs, output+"-go-tmp-")
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
