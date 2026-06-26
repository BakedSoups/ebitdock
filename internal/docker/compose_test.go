package docker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"

	"ebitdock/internal/config"
)

func TestGenerateComposeIncludesWebAndEnabledAPI(t *testing.T) {
	cfg := config.Config{
		Project: "demo",
		Docker:  config.DockerConfig{Enabled: true},
		Services: config.ServicesConfig{
			Web: config.ServiceConfig{
				Root:    "./static",
				Port:    8080,
				Image:   "nginx:1.27-alpine",
				Volumes: []string{"./static:/usr/share/nginx/html:ro"},
			},
			API: config.ServiceConfig{
				Enabled:   true,
				Command:   "go run ./server",
				Port:      3001,
				Image:     "golang:1.22",
				Workdir:   "/app",
				Env:       map[string]string{"PORT": "3001"},
				Volumes:   []string{".:/app"},
				DependsOn: []string{"database"},
			},
		},
	}
	cfg.SetDefaults()

	compose := GenerateCompose(cfg)
	web := compose.Services["web"]
	if web.Image != "nginx:1.27-alpine" {
		t.Fatalf("web image = %q", web.Image)
	}
	assertStrings(t, web.Ports, []string{"8080:8080"})
	assertStrings(t, web.Volumes, []string{"./static:/usr/share/nginx/html:ro"})

	api := compose.Services["api"]
	if api.Command != "go run ./server" || api.WorkingDir != "/app" {
		t.Fatalf("api service not populated: %+v", api)
	}
	if api.Environment["PORT"] != "3001" {
		t.Fatalf("api env not preserved: %+v", api.Environment)
	}
	assertStrings(t, api.Ports, []string{"3001:3001"})
	assertStrings(t, api.DependsOn, []string{"database"})
}

func TestGenerateComposeSkipsDisabledAPI(t *testing.T) {
	cfg := config.Config{Project: "demo"}
	cfg.SetDefaults()

	compose := GenerateCompose(cfg)
	if _, ok := compose.Services["api"]; ok {
		t.Fatal("disabled api should not be generated")
	}
	if _, ok := compose.Services["web"]; !ok {
		t.Fatal("web service should be generated")
	}
}

func TestGenerateComposeIncludesGenericServices(t *testing.T) {
	cfg := config.Config{
		Project: "demo",
		Services: config.ServicesConfig{
			Extra: map[string]config.ServiceConfig{
				"realtime": {
					Enabled:   true,
					Command:   "go run ./cmd/realtime",
					Port:      3002,
					Image:     "golang:1.22",
					Workdir:   "/app",
					Volumes:   []string{".:/app"},
					DependsOn: []string{"api"},
				},
				"admin": {
					Enabled: false,
					Command: "go run ./cmd/admin",
					Port:    9090,
				},
			},
		},
	}
	cfg.SetDefaults()

	compose := GenerateCompose(cfg)
	if _, ok := compose.Services["admin"]; ok {
		t.Fatal("disabled generic service should not be generated")
	}
	realtime, ok := compose.Services["realtime"]
	if !ok {
		t.Fatal("realtime service missing")
	}
	assertStrings(t, realtime.Ports, []string{"3002:3002"})
	assertStrings(t, realtime.DependsOn, []string{"api"})
}

func TestWriteComposeCreatesConfiguredFile(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Project: "demo",
		Docker:  config.DockerConfig{ComposeFile: ".ebitdock/compose.yaml"},
	}
	cfg.SetDefaults()

	path, err := WriteCompose(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, ".ebitdock", "compose.yaml") {
		t.Fatalf("path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ComposeFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "demo" {
		t.Fatalf("compose name = %q", parsed.Name)
	}
	if _, ok := parsed.Services["web"]; !ok {
		t.Fatalf("web service missing from %#v", parsed.Services)
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("strings = %#v, want %#v", got, want)
	}
}
