package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Project   string          `yaml:"project"`
	Game      GameConfig      `yaml:"game"`
	WASM      WASMConfig      `yaml:"wasm"`
	Services  ServicesConfig  `yaml:"services"`
	Dashboard DashboardConfig `yaml:"dashboard"`
	Watch     WatchConfig     `yaml:"watch"`

	// Legacy fields kept so older ebitdock.yaml files still load.
	Web    WebConfig    `yaml:"web"`
	Server ServerConfig `yaml:"server"`
}

type GameConfig struct {
	Package string `yaml:"package"`
	Output  string `yaml:"output"`
}

type WebConfig struct {
	Root          string `yaml:"root"`
	Port          int    `yaml:"port"`
	DashboardPort int    `yaml:"dashboard_port"`
}

type WASMConfig struct {
	Exec string `yaml:"exec"`
}

type ServicesConfig struct {
	Web ServiceConfig `yaml:"web"`
	API ServiceConfig `yaml:"api"`
}

type ServiceConfig struct {
	Enabled bool   `yaml:"enabled"`
	Command string `yaml:"command"`
	Root    string `yaml:"root"`
	Port    int    `yaml:"port"`
}

type DashboardConfig struct {
	Port int `yaml:"port"`
}

type ServerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Command string `yaml:"command"`
	Port    int    `yaml:"port"`
}

type WatchConfig struct {
	Rebuild []string `yaml:"rebuild"`
	Static  []string `yaml:"static"`
}

func (w *WatchConfig) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var legacy []string
		if err := value.Decode(&legacy); err != nil {
			return err
		}
		w.Rebuild = legacy
		return nil
	case yaml.MappingNode:
		type raw WatchConfig
		return value.Decode((*raw)(w))
	default:
		return nil
	}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) SetDefaults() {
	if c.Game.Package == "" {
		c.Game.Package = "./game"
	}
	if c.Game.Output == "" {
		c.Game.Output = "./static/game.wasm"
	}
	if c.WASM.Exec == "" {
		c.WASM.Exec = "./static/wasm_exec.js"
	}
	if c.Services.Web.Root == "" {
		c.Services.Web.Root = c.Web.Root
	}
	if c.Services.Web.Root == "" {
		c.Services.Web.Root = "./static"
	}
	if c.Services.Web.Port == 0 {
		c.Services.Web.Port = c.Web.Port
	}
	if c.Services.Web.Port == 0 {
		c.Services.Web.Port = 8080
	}
	if c.Dashboard.Port == 0 {
		c.Dashboard.Port = c.Web.DashboardPort
	}
	if c.Dashboard.Port == 0 {
		c.Dashboard.Port = c.Services.Web.Port + 1
	}
	if c.Services.API.Command == "" {
		c.Services.API.Command = c.Server.Command
	}
	if c.Services.API.Command == "" {
		c.Services.API.Command = "go run ./server"
	}
	if !c.Services.API.Enabled {
		c.Services.API.Enabled = c.Server.Enabled
	}
	if c.Services.API.Port == 0 {
		c.Services.API.Port = c.Server.Port
	}
	if c.Services.API.Port == 0 {
		c.Services.API.Port = 3001
	}
	if len(c.Watch.Rebuild) == 0 {
		c.Watch.Rebuild = []string{"./game/**/*.go", "./assets/**"}
	}
	if len(c.Watch.Static) == 0 {
		c.Watch.Static = []string{c.Services.Web.Root + "/**"}
	}
}

func (c Config) Validate() error {
	if c.Project == "" {
		return fmt.Errorf("project is required in ebitdock.yaml")
	}
	if c.Game.Package == "" || c.Game.Output == "" || c.Services.Web.Root == "" {
		return fmt.Errorf("game.package, game.output, and services.web.root are required")
	}
	return nil
}

func (c Config) StaticRoot() string {
	return c.Services.Web.Root
}

func (c Config) WebPort() int {
	return c.Services.Web.Port
}

func (c Config) DashboardPort() int {
	return c.Dashboard.Port
}

func (c Config) APIEnabled() bool {
	return c.Services.API.Enabled
}

func (c Config) APICommand() string {
	return c.Services.API.Command
}

func (c Config) APIPort() int {
	return c.Services.API.Port
}

func (c Config) WASMExecPath() string {
	return c.WASM.Exec
}

func (c Config) RebuildWatch() []string {
	return append([]string(nil), c.Watch.Rebuild...)
}

func (c Config) StaticWatch() []string {
	return append([]string(nil), c.Watch.Static...)
}

func (c Config) WatchPatterns() []string {
	patterns := append([]string(nil), c.Watch.Rebuild...)
	patterns = append(patterns, c.Watch.Static...)
	return patterns
}
