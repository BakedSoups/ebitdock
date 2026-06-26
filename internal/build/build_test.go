package build

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ebitdock/internal/config"
)

func TestDockerBuildCommandUsesGoImageAndProjectMount(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Project: "demo",
		Docker:  config.DockerConfig{Enabled: true, GoImage: "golang:1.25"},
		Game:    config.GameConfig{Package: "./cmd/game", Output: "./static/game.wasm"},
		WASM:    config.WASMConfig{Exec: "./static/wasm_exec.js"},
	}
	cfg.SetDefaults()

	name, args, err := dockerBuildCommand(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if name != "docker" {
		t.Fatalf("name = %q, want docker", name)
	}
	wantPrefix := []string{
		"run", "--rm",
		"-v", root + ":/app",
		"-w", "/app",
		"-e", "GOOS=js",
		"-e", "GOARCH=wasm",
		"golang:1.25",
		"sh", "-c",
	}
	if !reflect.DeepEqual(args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("args prefix = %#v, want %#v", args[:len(wantPrefix)], wantPrefix)
	}
	script := args[len(args)-1]
	for _, want := range []string{"go build -mod=mod", "/app/static/game.wasm", "./cmd/game", "/app/static/wasm_exec.js"} {
		if !strings.Contains(script, want) {
			t.Fatalf("script %q does not contain %q", script, want)
		}
	}
}

func TestDockerBuildCommandUsesNestedModuleWorkdir(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "game")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, "go.mod"), []byte("module example.com/game\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Project: "demo",
		Docker:  config.DockerConfig{Enabled: true, GoImage: "golang:1.25"},
		Game:    config.GameConfig{Package: "./game", Output: "./static/game.wasm"},
		WASM:    config.WASMConfig{Exec: "./static/wasm_exec.js"},
	}
	cfg.SetDefaults()

	_, args, err := dockerBuildCommand(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if args[5] != "/app/game" {
		t.Fatalf("workdir = %q, want /app/game", args[5])
	}
	if !strings.Contains(args[len(args)-1], "go build -mod=mod -o '/app/static/game.wasm' '.'") {
		t.Fatalf("script did not build nested module from dot: %q", args[len(args)-1])
	}
}
