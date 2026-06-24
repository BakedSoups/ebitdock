package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"ebitdock/internal/build"
	"ebitdock/internal/config"
	"ebitdock/internal/dev"
	"ebitdock/internal/process"
	"ebitdock/internal/templates"
)

const usage = `ebitdock manages the web shell around an Ebitengine WASM game.

Usage:
  ebitdock init [name|.]
  ebitdock dev
  ebitdock build wasm
  ebitdock logs
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ebitdock: %v\n", err)
		os.Exit(1)
	}
}

// run is intentionally thin: command parsing lives here, while command behavior
// stays in internal packages that can be tested without a CLI process.
func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "init":
		if len(args) > 2 {
			return errors.New("usage: ebitdock init [name|.]")
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
		}
		return templates.InitProject(name)
	case "dev":
		cfg, root, err := loadProject()
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return dev.Run(ctx, root, cfg)
	case "build":
		if len(args) != 2 || args[1] != "wasm" {
			return errors.New("usage: ebitdock build wasm")
		}
		cfg, root, err := loadProject()
		if err != nil {
			return err
		}
		status := newStatus(root, cfg)
		result := build.WASM(context.Background(), root, cfg, status)
		fmt.Print(result.Log)
		if result.Err != nil {
			return result.Err
		}
		fmt.Printf("built %s in %s\n", cfg.Game.Output, result.Duration)
		return nil
	case "logs":
		_, root, err := loadProject()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(logPath(root))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], strings.TrimSpace(usage))
	}
}

// newStatus creates the shared dashboard/log state for commands that operate on
// a loaded project. Persistent logs are per project root, not global.
func newStatus(root string, cfg config.Config) *process.Status {
	status := process.NewStatus(cfg)
	status.SetLogFile(logPath(root))
	return status
}

func logPath(root string) string {
	return filepath.Join(root, ".ebitdock", "ebitdock.log")
}

// loadProject treats the current working directory as the project root. This
// keeps ebitdock close to Go tooling: cd into the project, then run commands.
func loadProject() (config.Config, string, error) {
	root, err := filepath.Abs(".")
	if err != nil {
		return config.Config{}, "", err
	}
	cfg, err := config.Load(filepath.Join(root, "ebitdock.yaml"))
	if err != nil {
		return config.Config{}, "", err
	}
	return cfg, root, nil
}
