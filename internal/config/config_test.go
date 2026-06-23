package config

import "testing"

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
