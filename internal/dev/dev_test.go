package dev

import (
	"path/filepath"
	"testing"

	"github.com/BakedSoups/ebitdock/internal/config"
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
