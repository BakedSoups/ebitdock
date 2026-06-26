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
