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
	"github.com/BakedSoups/ebitdock/internal/tools"
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
	var wasmserve *process.Command
	if cfg.WASMDevServer() == "wasmserve" {
		cmd, err := startWasmserve(ctx, root, cfg, status)
		if err != nil {
			return err
		}
		wasmserve = cmd
		defer wasmserve.Stop()
	}
	composePath, err := dock.WriteCompose(root, cfg)
	if err != nil {
		return err
	}
	status.AppendLog("wrote compose file: " + composePath)

	if cfg.WASMDevServer() != "wasmserve" {
		if err := checkedBuild(ctx, root, cfg, status, os.Stdout); err != nil {
			return err
		}
	} else {
		status.SetBuild("wasmserve", "", nil)
		status.AppendLog("wasmserve owns WASM rebuilds")
		for _, hint := range tools.BrowserShellHints(root, cfg.StaticRoot()) {
			status.AppendLog("dev hint: " + hint)
			fmt.Fprintln(os.Stdout, "warn\t"+hint)
		}
	}

	var stack *process.Command
	if len(cfg.EnabledServices()) > 0 {
		name, args, err := dock.ComposeCommand(cfg.ComposeFile(), "up", "--build", "--remove-orphans")
		if err != nil {
			return err
		}

		var setBackend func(string)
		if cfg.APIEnabled() {
			setBackend = status.SetServer
		}
		stack, err = process.StartArgs(ctx, root, name, args, "docker", status, setBackend)
		if err != nil {
			return err
		}
		status.SetServices("running")
	}
	defer func() {
		if stack != nil {
			stack.Stop()
			status.SetServices("stopped")
			dockerComposeDown(root, cfg, status)
		}
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
	webService := "docker"
	if cfg.WASMDevServer() == "wasmserve" {
		webService = "wasmserve"
	}
	cliui.DevStatus(os.Stdout, cfg, webService)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case err := <-commandDone(stack):
			if err != nil {
				return err
			}
			return nil
		case err := <-commandDone(wasmserve):
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
			if cfg.WASMDevServer() == "wasmserve" {
				status.AppendLog("source file changed; notifying wasmserve")
				if err := tools.NotifyWasmserve(ctx, cfg.WebPort()); err != nil {
					status.AppendLog("wasmserve notify failed: " + err.Error())
					fmt.Fprintln(os.Stdout, "notify\tfailed\t"+err.Error())
				} else {
					status.AppendLog("wasmserve notified")
					fmt.Fprintln(os.Stdout, "notify\tok")
				}
			} else {
				if err := checkedBuild(ctx, root, cfg, status, os.Stdout); err != nil {
					continue
				}
			}
		}
	}
}

func startWasmserve(ctx context.Context, root string, cfg config.Config, status *process.Status) (*process.Command, error) {
	if err := tools.CheckWasmserve(nil); err != nil {
		return nil, err
	}
	dir, target, err := wasmserveWorkingDirAndTarget(root, cfg)
	if err != nil {
		return nil, err
	}
	name, args, err := tools.WasmserveCommand(cfg.WebPort(), target)
	if err != nil {
		return nil, err
	}
	cmd, err := process.StartArgs(ctx, dir, name, args, "wasmserve", status, nil)
	if err != nil {
		return nil, err
	}
	status.AppendLog("wasmserve started")
	return cmd, nil
}

func commandDone(cmd *process.Command) <-chan error {
	if cmd == nil {
		return nil
	}
	return cmd.Done()
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

func wasmserveWorkingDirAndTarget(root string, cfg config.Config) (dir, target string, err error) {
	staticDir, err := filepath.Abs(filepath.Join(root, cfg.StaticRoot()))
	if err != nil {
		return "", "", err
	}
	if info, err := os.Stat(staticDir); err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("services.web.root %q does not exist\nresolved path: %s\ncreate it or update services.web.root in ebitdock.yaml", cfg.StaticRoot(), staticDir)
		}
		return "", "", fmt.Errorf("check services.web.root %q at %s: %w", cfg.StaticRoot(), staticDir, err)
	} else if !info.IsDir() {
		return "", "", fmt.Errorf("services.web.root %q is not a directory\nresolved path: %s\nupdate services.web.root in ebitdock.yaml", cfg.StaticRoot(), staticDir)
	}

	if !isLocalPackage(cfg.Game.Package) {
		return staticDir, cfg.Game.Package, nil
	}
	gameDir, err := filepath.Abs(filepath.Join(root, cfg.Game.Package))
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(staticDir, gameDir)
	if err != nil {
		return "", "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return staticDir, ".", nil
	}
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return staticDir, rel, nil
}

func isLocalPackage(pkg string) bool {
	return pkg == "." || strings.HasPrefix(pkg, "./") || strings.HasPrefix(pkg, "../") || filepath.IsAbs(pkg)
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
