package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferGamePackagePrefersCmdGame(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "game"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "game", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := inferGamePackage(root, "nongrampictures")
	if got != "./cmd/game" {
		t.Fatalf("inferGamePackage() = %q, want ./cmd/game", got)
	}
}

func TestInferGamePackagePrefersNamedCommand(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "mygame"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "mygame", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "game"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "game", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := inferGamePackage(root, "mygame")
	if got != "./cmd/mygame" {
		t.Fatalf("inferGamePackage() = %q, want ./cmd/mygame", got)
	}
}

func TestInitCurrentProjectWritesOnlyConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "game"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "game", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "index.html"), []byte("owned by app\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := InitCurrentProject("demo"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "ebitdock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "package: ./cmd/game") {
		t.Fatalf("config did not infer cmd/game:\n%s", data)
	}
	if !strings.Contains(string(data), "root: ./static") {
		t.Fatalf("config did not infer static root:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(root, "web", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("init should not generate web shell for existing repos")
	}
}
