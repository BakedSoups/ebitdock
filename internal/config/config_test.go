package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaults(t *testing.T) {
	cfg := Config{Project: "demo"}
	cfg.SetDefaults()
	if cfg.WebPort() != 8080 {
		t.Fatalf("WebPort() = %d, want 8080", cfg.WebPort())
	}
	if cfg.DashboardPort() != 8081 {
		t.Fatalf("DashboardPort() = %d, want 8081", cfg.DashboardPort())
	}
	if cfg.Game.Output == "" || cfg.WASMExecPath() == "" || len(cfg.WatchPatterns()) == 0 {
		t.Fatalf("defaults were not populated: %+v", cfg)
	}
	if cfg.ComposeFile() != ".ebitdock/compose.yaml" {
		t.Fatalf("ComposeFile() = %q, want .ebitdock/compose.yaml", cfg.ComposeFile())
	}
	if cfg.GoImage() != "golang:1.24" {
		t.Fatalf("GoImage() = %q, want golang:1.24", cfg.GoImage())
	}
	if cfg.Services.Web.Image != "nginx:1.27-alpine" {
		t.Fatalf("web image = %q, want nginx:1.27-alpine", cfg.Services.Web.Image)
	}
}

func TestWebCommand(t *testing.T) {
	cfg := Config{
		Project: "demo",
		Services: ServicesConfig{
			Web: ServiceConfig{Command: "go run ."},
		},
	}
	cfg.SetDefaults()
	if !cfg.UsesWebCommand() {
		t.Fatal("UsesWebCommand() = false, want true")
	}
	if cfg.WebCommand() != "go run ." {
		t.Fatalf("WebCommand() = %q, want go run .", cfg.WebCommand())
	}
}

func TestDockerServiceFieldsArePreserved(t *testing.T) {
	cfg := Config{
		Project: "demo",
		Docker:  DockerConfig{Enabled: true, ComposeFile: "./compose.dev.yaml", GoImage: "golang:1.25"},
		Services: ServicesConfig{
			Web: ServiceConfig{
				Image:   "caddy:2",
				Workdir: "/srv",
				Volumes: []string{"./public:/srv:ro"},
				Env:     map[string]string{"MODE": "dev"},
			},
		},
	}
	cfg.SetDefaults()
	if !cfg.DockerEnabled() {
		t.Fatal("DockerEnabled() = false, want true")
	}
	if cfg.ComposeFile() != "./compose.dev.yaml" {
		t.Fatalf("ComposeFile() = %q, want ./compose.dev.yaml", cfg.ComposeFile())
	}
	if cfg.Services.Web.Image != "caddy:2" || cfg.Services.Web.Workdir != "/srv" {
		t.Fatalf("web docker fields not preserved: %+v", cfg.Services.Web)
	}
	assertStrings(t, cfg.Services.Web.Volumes, []string{"./public:/srv:ro"})
	if cfg.Services.Web.Env["MODE"] != "dev" {
		t.Fatalf("web env not preserved: %+v", cfg.Services.Web.Env)
	}
}

func TestGenericServicesArePreservedAndDefaulted(t *testing.T) {
	data := []byte(`project: demo
services:
  web:
    root: ./static
    port: 8080
  realtime:
    enabled: true
    command: go run ./cmd/realtime
    port: 3002
    depends_on:
      - api
  database:
    enabled: true
    kind: postgres
    image: postgres:16-alpine
    port: 5432
`)
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.SetDefaults()

	services := cfg.EnabledServices()
	realtime, ok := services["realtime"]
	if !ok {
		t.Fatal("realtime service missing")
	}
	if realtime.Kind != "go" || realtime.Image != "golang:1.24" || realtime.Workdir != "/app" {
		t.Fatalf("realtime defaults not applied: %+v", realtime)
	}
	assertStrings(t, realtime.DependsOn, []string{"api"})
	assertPorts(t, realtime.Ports, []PortConfig{{Name: "realtime", Port: 3002, URL: "http://localhost:3002"}})

	database, ok := services["database"]
	if !ok {
		t.Fatal("database service missing")
	}
	if database.Kind != "postgres" || database.Image != "postgres:16-alpine" {
		t.Fatalf("database not preserved: %+v", database)
	}
	assertPorts(t, database.Ports, []PortConfig{{Name: "database", Port: 5432, URL: "http://localhost:5432"}})
}

func TestBeforeRebuildCheckDefaultUsesGamePackage(t *testing.T) {
	cfg := Config{
		Project: "demo",
		Game:    GameConfig{Package: "./wasm"},
	}
	cfg.SetDefaults()
	if cfg.BeforeRebuildCheckEnabled() {
		t.Fatal("BeforeRebuildCheckEnabled() = true, want false")
	}
	if cfg.BeforeRebuildCheckCommand() != "go test ./wasm" {
		t.Fatalf("BeforeRebuildCheckCommand() = %q, want go test ./wasm", cfg.BeforeRebuildCheckCommand())
	}
}

func TestServicePortsIncludePrimaryAndDedupeExtras(t *testing.T) {
	cfg := Config{
		Project: "demo",
		Services: ServicesConfig{
			Web: ServiceConfig{
				Port:  8080,
				Ports: []PortConfig{{Port: 8080}, {Name: "admin", Port: 3000}, {Port: 9090}, {Port: 3000}},
			},
			API: ServiceConfig{
				Port:  3001,
				Ports: []PortConfig{{Port: 3001}, {Name: "socket", Port: 3002, URL: "ws://localhost:3002"}},
			},
		},
	}
	cfg.SetDefaults()

	assertPorts(t, cfg.WebPorts(), []PortConfig{
		{Name: "web", Port: 8080, URL: "http://localhost:8080"},
		{Name: "admin", Port: 3000, URL: "http://localhost:3000"},
		{Name: "port 9090", Port: 9090, URL: "http://localhost:9090"},
	})
	assertPorts(t, cfg.APIPorts(), []PortConfig{
		{Name: "api", Port: 3001, URL: "http://localhost:3001"},
		{Name: "socket", Port: 3002, URL: "ws://localhost:3002"},
	})
}

func TestPortConfigUnmarshalAcceptsNumberAndMapping(t *testing.T) {
	var cfg struct {
		Ports []PortConfig `yaml:"ports"`
	}
	data := []byte(`ports:
  - 8080
  - name: sockets
    port: 3002
    url: ws://localhost:3002
`)
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	want := []PortConfig{
		{Port: 8080},
		{Name: "sockets", Port: 3002, URL: "ws://localhost:3002"},
	}
	if !reflect.DeepEqual(cfg.Ports, want) {
		t.Fatalf("ports = %#v, want %#v", cfg.Ports, want)
	}
}

func assertPorts(t *testing.T, got, want []PortConfig) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ports = %#v, want %#v", got, want)
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("strings = %#v, want %#v", got, want)
	}
}
