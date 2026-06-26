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

	"ebitdock/internal/build"
	"ebitdock/internal/checks"
	"ebitdock/internal/cliui"
	"ebitdock/internal/config"
	"ebitdock/internal/dashboard"
	dock "ebitdock/internal/docker"
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
	if cfg.DockerEnabled() {
		return runDocker(ctx, root, cfg, status)
	}

	web, webMode, err := startWeb(ctx, root, cfg, status)
	if err != nil {
		return err
	}
	defer web.Stop()

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
	cliui.DevStatus(os.Stdout, cfg, webServiceName(webMode))
	if webMode == webModeWasmserve {
		for _, hint := range tools.BrowserShellHints(root, cfg.StaticRoot()) {
			status.AppendLog("dev hint: " + hint)
			fmt.Fprintln(os.Stdout, "warn\t"+hint)
		}
	} else {
		if err := checkedBuild(ctx, root, cfg, status, os.Stdout); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case err := <-web.Done():
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
			fmt.Fprintln(os.Stdout, "change\t"+path)
			if isStaticSourceChange(root, cfg, path) {
				status.AppendLog("static file changed")
			} else {
				if webMode == webModeCommand {
					if err := checkedBuild(ctx, root, cfg, status, os.Stdout); err != nil {
						continue
					}
				} else {
					status.AppendLog("source file changed; notifying wasmserve")
				}
			}
			if webMode == webModeWasmserve {
				if err := tools.NotifyWasmserve(ctx, cfg.WebPort()); err != nil {
					status.AppendLog("wasmserve notify failed: " + err.Error())
					fmt.Fprintln(os.Stdout, "notify\tfailed\t"+err.Error())
				} else {
					status.AppendLog("wasmserve notified")
					fmt.Fprintln(os.Stdout, "notify\tok")
				}
			}
		}
	}
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
	defer func() {
		stack.Stop()
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

type webMode string

const (
	webModeCommand   webMode = "command"
	webModeWasmserve webMode = "wasmserve"
)

func startWeb(ctx context.Context, root string, cfg config.Config, status *process.Status) (*process.Command, webMode, error) {
	if cfg.UsesWebCommand() {
		cmd, err := process.StartCommand(ctx, root, cfg.WebCommand(), "web", status, nil)
		return cmd, webModeCommand, err
	}
	if err := tools.CheckWasmserve(nil); err != nil {
		return nil, "", err
	}
	wasmserveDir, wasmserveTarget, err := wasmserveWorkingDirAndTarget(root, cfg)
	if err != nil {
		return nil, "", err
	}
	wasmserveName, wasmserveArgs, err := tools.WasmserveCommand(cfg.WebPort(), wasmserveTarget)
	if err != nil {
		return nil, "", err
	}
	cmd, err := process.StartArgs(ctx, wasmserveDir, wasmserveName, wasmserveArgs, "wasmserve", status, nil)
	return cmd, webModeWasmserve, err
}

func printBuildResult(out io.Writer, result build.Result) {
	if result.Err != nil {
		fmt.Fprintln(out, "build\tfailed\t"+result.Err.Error())
		return
	}
	fmt.Fprintln(out, "build\tok\t"+result.Duration.String())
}

func checkedBuild(ctx context.Context, root string, cfg config.Config, status *process.Status, out io.Writer) error {
	if cfg.BeforeRebuildCheckEnabled() {
		result := checks.Run(ctx, root, cfg.BeforeRebuildCheckCommand(), status)
		printCheckResult(out, result)
		if result.Err != nil {
			return result.Err
		}
	}
	status.AppendLog("rebuilding wasm")
	result := build.WASM(ctx, root, cfg, status)
	printBuildResult(out, result)
	return result.Err
}

func printCheckResult(out io.Writer, result checks.Result) {
	if result.Err != nil {
		fmt.Fprintln(out, "check\tfailed\t"+result.Err.Error())
		return
	}
	fmt.Fprintln(out, "check\tok\t"+result.Duration.String())
}

func webServiceName(mode webMode) string {
	if mode == webModeCommand {
		return "web"
	}
	return "wasmserve"
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
