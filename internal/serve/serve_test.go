package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/BakedSoups/ebitdock/internal/config"
)

func TestServerServesWASMWithCorrectContentType(t *testing.T) {
	root := t.TempDir()
	staticRoot := filepath.Join(root, "static")
	if err := os.MkdirAll(staticRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticRoot, "game.wasm"), []byte("wasm"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Services: config.ServicesConfig{Web: config.ServiceConfig{Root: "./static"}}}
	cfg.SetDefaults()

	req := httptest.NewRequest(http.MethodGet, "/game.wasm", nil)
	rec := httptest.NewRecorder()
	Server(root, cfg).ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "application/wasm" {
		t.Fatalf("Content-Type = %q, want application/wasm", got)
	}
}
