package dev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ebitdock/internal/config"
)

func TestIsStaticSourceChangeOnlyMatchesFilesUnderStaticRoot(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{Services: config.ServicesConfig{Web: config.ServiceConfig{Root: "./static"}}}
	cfg.SetDefaults()

	if !isStaticSourceChange(root, cfg, filepath.Join(root, "static", "index.html")) {
		t.Fatal("static/index.html should be treated as a static source change")
	}
	if isStaticSourceChange(root, cfg, filepath.Join(root, "internal", "game", "render.go")) {
		t.Fatal("internal/game/render.go should not be treated as a static source change")
	}
}

func TestWasmserveWorkingDirAndTargetUsesStaticRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "game"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Project: "demo",
		Game:    config.GameConfig{Package: "./cmd/game"},
		Services: config.ServicesConfig{
			Web: config.ServiceConfig{Root: "./static"},
		},
	}
	cfg.SetDefaults()

	dir, target, err := wasmserveWorkingDirAndTarget(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(root, "static") {
		t.Fatalf("dir = %q, want static root", dir)
	}
	if target != "../cmd/game" {
		t.Fatalf("target = %q, want ../cmd/game", target)
	}
}

func TestWasmserveWorkingDirAndTargetKeepsImportPathTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Project: "demo",
		Game:    config.GameConfig{Package: "example.com/game/cmd/game"},
		Services: config.ServicesConfig{
			Web: config.ServiceConfig{Root: "./public"},
		},
	}
	cfg.SetDefaults()

	dir, target, err := wasmserveWorkingDirAndTarget(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(root, "public") {
		t.Fatalf("dir = %q, want public root", dir)
	}
	if target != "example.com/game/cmd/game" {
		t.Fatalf("target = %q, want import path", target)
	}
}

func TestWasmserveWorkingDirAndTargetExplainsMissingStaticRoot(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Project: "demo",
		Game:    config.GameConfig{Package: "./cmd/game"},
		Services: config.ServicesConfig{
			Web: config.ServiceConfig{Root: "./static"},
		},
	}
	cfg.SetDefaults()

	_, _, err := wasmserveWorkingDirAndTarget(root, cfg)
	if err == nil {
		t.Fatal("expected missing static root error")
	}
	msg := err.Error()
	for _, want := range []string{"services.web.root", "./static", "resolved path", "create it or update services.web.root"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q did not contain %q", msg, want)
		}
	}
}
