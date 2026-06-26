package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the in-memory form of ebitdock.yaml. The top-level shape is meant
// to read like a small compose file: game build inputs, services, dashboard,
// and file-watch groups.
type Config struct {
	Project   string          `yaml:"project"`
	Game      GameConfig      `yaml:"game"`
	WASM      WASMConfig      `yaml:"wasm"`
	Docker    DockerConfig    `yaml:"docker"`
	Services  ServicesConfig  `yaml:"services"`
	Checks    ChecksConfig    `yaml:"checks"`
	Dashboard DashboardConfig `yaml:"dashboard"`
	Watch     WatchConfig     `yaml:"watch"`

	// Legacy fields kept so older ebitdock.yaml files still load.
	Web    WebConfig    `yaml:"web"`
	Server ServerConfig `yaml:"server"`
}

// GameConfig points at the Go package to build and the output WASM file for
// explicit production-style builds.
type GameConfig struct {
	Package string `yaml:"package"`
	Output  string `yaml:"output"`
}

// WebConfig is the old pre-services web block. It is still decoded so projects
// created by earlier ebitdock versions do not fail immediately.
type WebConfig struct {
	Root          string `yaml:"root"`
	Port          int    `yaml:"port"`
	DashboardPort int    `yaml:"dashboard_port"`
}

// WASMConfig controls where the Go runtime shim is copied during build wasm.
type WASMConfig struct {
	Exec string `yaml:"exec"`
}

// DockerConfig controls how ebitdock generates and runs docker compose files.
type DockerConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Mode        string `yaml:"mode"`
	ComposeFile string `yaml:"compose_file"`
	GoImage     string `yaml:"go_image"`

	enabledSet bool
}

// UnmarshalYAML keeps docker.enabled working for older configs while allowing
// the newer docker.mode shape to avoid a top-level feature boolean.
func (d *DockerConfig) UnmarshalYAML(value *yaml.Node) error {
	type raw DockerConfig
	var decoded raw
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value == "enabled" {
			decoded.enabledSet = true
			break
		}
	}
	*d = DockerConfig(decoded)
	return nil
}

// ChecksConfig contains optional commands that gate dev/build workflows.
type ChecksConfig struct {
	BeforeRebuild CheckConfig `yaml:"before_rebuild"`
}

// CheckConfig describes one optional command check.
type CheckConfig struct {
	Enabled bool   `yaml:"enabled"`
	Command string `yaml:"command"`
}

// ServicesConfig groups local processes that dev mode can start and track.
type ServicesConfig struct {
	Web   ServiceConfig            `yaml:"web"`
	API   ServiceConfig            `yaml:"api"`
	Extra map[string]ServiceConfig `yaml:"-"`
}

// ServiceConfig is shared by static web service and optional API process. Some
// fields are only meaningful for one service type.
type ServiceConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Kind       string            `yaml:"kind"`
	Command    string            `yaml:"command"`
	Root       string            `yaml:"root"`
	Port       int               `yaml:"port"`
	Ports      []PortConfig      `yaml:"ports"`
	Image      string            `yaml:"image"`
	Dockerfile string            `yaml:"dockerfile"`
	Workdir    string            `yaml:"workdir"`
	Env        map[string]string `yaml:"env"`
	Volumes    []string          `yaml:"volumes"`
	DependsOn  []string          `yaml:"depends_on"`
}

// PortConfig describes one dashboard-visible port exposed by a service.
type PortConfig struct {
	Name string `yaml:"name" json:"name"`
	Port int    `yaml:"port" json:"port"`
	URL  string `yaml:"url" json:"url"`
}

// UnmarshalYAML accepts both compact numeric ports and named port mappings:
// ports: [8080, 3001] or ports: [{name: api, port: 3001}].
func (p *PortConfig) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		port, err := strconv.Atoi(value.Value)
		if err != nil {
			return fmt.Errorf("port must be a number: %q", value.Value)
		}
		p.Port = port
		return nil
	case yaml.MappingNode:
		type raw PortConfig
		return value.Decode((*raw)(p))
	default:
		return fmt.Errorf("port must be a number or mapping")
	}
}

// DashboardConfig keeps the dashboard port separate from the user-facing web
// service port.
type DashboardConfig struct {
	Port int `yaml:"port"`
}

// ServerConfig is the old backend block. It maps into services.api defaults.
type ServerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Command string `yaml:"command"`
	Port    int    `yaml:"port"`
}

// WatchConfig separates source changes that should rebuild WASM from static
// changes that only need to be logged or handled by the user's browser tooling.
type WatchConfig struct {
	Rebuild []string `yaml:"rebuild"`
	Static  []string `yaml:"static"`
}

// UnmarshalYAML accepts both the new mapping form and the older list form:
// watch: [./game/**/*.go, ./assets/**]. Old lists become rebuild watches.
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

// UnmarshalYAML keeps compatibility with the original fixed web/api model
// while preserving arbitrary named services for Docker Compose projects.
func (s *ServicesConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]ServiceConfig
	if err := value.Decode(&raw); err != nil {
		return err
	}
	s.Extra = map[string]ServiceConfig{}
	for name, service := range raw {
		switch name {
		case "web":
			s.Web = service
		case "api":
			s.API = service
		default:
			s.Extra[name] = service
		}
	}
	return nil
}

// Load reads ebitdock.yaml, applies compatibility defaults, and validates the
// normalized config used by the rest of the program.
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

// SetDefaults normalizes old and new config shapes into the services-based
// model. Prefer helper methods like WebPort over reading raw fields directly.
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
	if c.Docker.ComposeFile == "" {
		c.Docker.ComposeFile = ".ebitdock/compose.yaml"
	}
	if c.Docker.GoImage == "" {
		c.Docker.GoImage = "golang:1.24"
	}
	if c.Checks.BeforeRebuild.Command == "" {
		c.Checks.BeforeRebuild.Command = "go test " + c.Game.Package
	}
	if c.Services.Web.Root == "" {
		c.Services.Web.Root = c.Web.Root
	}
	if c.Services.Web.Root == "" {
		c.Services.Web.Root = "./static"
	}
	if c.Services.Web.Kind == "" {
		c.Services.Web.Kind = "static"
	}
	if c.Services.Web.Image == "" {
		c.Services.Web.Image = "nginx:1.27-alpine"
	}
	if c.Services.Web.Workdir == "" {
		c.Services.Web.Workdir = "/usr/share/nginx/html"
	}
	if len(c.Services.Web.Volumes) == 0 {
		c.Services.Web.Volumes = []string{c.Services.Web.Root + ":/usr/share/nginx/html:ro"}
	}
	if c.Services.Web.Port == 0 {
		c.Services.Web.Port = c.Web.Port
	}
	if c.Services.Web.Port == 0 {
		c.Services.Web.Port = 8080
	}
	c.Services.Web.Ports = normalizePorts("web", c.Services.Web.Port, c.Services.Web.Ports)
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
	if c.Services.API.Kind == "" {
		c.Services.API.Kind = "go"
	}
	if c.Services.API.Image == "" {
		c.Services.API.Image = c.Docker.GoImage
	}
	if c.Services.API.Workdir == "" {
		c.Services.API.Workdir = "/app"
	}
	if len(c.Services.API.Volumes) == 0 {
		c.Services.API.Volumes = []string{".:/app"}
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
	c.Services.API.Ports = normalizePorts("api", c.Services.API.Port, c.Services.API.Ports)
	if c.Services.Extra == nil {
		c.Services.Extra = map[string]ServiceConfig{}
	}
	for name, service := range c.Services.Extra {
		service = c.defaultNamedService(name, service)
		c.Services.Extra[name] = service
	}
	if len(c.Watch.Rebuild) == 0 {
		c.Watch.Rebuild = []string{"./game/**/*.go", "./assets/**"}
	}
	if len(c.Watch.Static) == 0 {
		c.Watch.Static = []string{c.Services.Web.Root + "/**"}
	}
}

func (c Config) defaultNamedService(name string, service ServiceConfig) ServiceConfig {
	if service.Kind == "" {
		if service.Image != "" && service.Command == "" {
			service.Kind = "custom"
		} else {
			service.Kind = "go"
		}
	}
	if service.Port != 0 {
		service.Ports = normalizePorts(name, service.Port, service.Ports)
	}
	if service.Kind == "go" || service.Command != "" {
		if service.Image == "" {
			service.Image = c.Docker.GoImage
		}
		if service.Workdir == "" {
			service.Workdir = "/app"
		}
		if len(service.Volumes) == 0 {
			service.Volumes = []string{".:/app"}
		}
	}
	return service
}

func normalizePorts(primaryName string, primary int, extras []PortConfig) []PortConfig {
	seen := map[int]bool{}
	var ports []PortConfig
	if primary != 0 {
		seen[primary] = true
		ports = append(ports, normalizePort(PortConfig{Name: primaryName, Port: primary}))
	}
	for _, port := range extras {
		port = normalizePort(port)
		if port.Port == 0 || seen[port.Port] {
			continue
		}
		seen[port.Port] = true
		ports = append(ports, port)
	}
	return ports
}

func normalizePort(port PortConfig) PortConfig {
	port.Name = strings.TrimSpace(port.Name)
	if port.Name == "" && port.Port != 0 {
		port.Name = "port " + strconv.Itoa(port.Port)
	}
	if port.URL == "" && port.Port != 0 {
		port.URL = "http://localhost:" + strconv.Itoa(port.Port)
	}
	return port
}

// Validate checks only the values needed for command execution. Deeper checks,
// like whether a package exists, are left to the command that uses them.
func (c Config) Validate() error {
	if c.Project == "" {
		return fmt.Errorf("project is required in ebitdock.yaml")
	}
	if c.Game.Package == "" || c.Game.Output == "" || c.Services.Web.Root == "" {
		return fmt.Errorf("game.package, game.output, and services.web.root are required")
	}
	return nil
}

// DockerEnabled reports whether dev should use docker compose for services.
func (c Config) DockerEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(c.Docker.Mode)) {
	case "local", "wasmserve", "off", "disabled":
		return false
	case "docker", "compose", "":
	default:
		return true
	}
	if c.Docker.enabledSet {
		return c.Docker.Enabled
	}
	if c.Docker.Enabled {
		return true
	}
	return true
}

// ComposeFile returns the project-local docker compose file ebitdock manages.
func (c Config) ComposeFile() string {
	return c.Docker.ComposeFile
}

// GoImage returns the Go image used for containerized tool commands.
func (c Config) GoImage() string {
	return c.Docker.GoImage
}

// StaticRoot returns the directory served as the browser app.
func (c Config) StaticRoot() string {
	return c.Services.Web.Root
}

// WebCommand returns the project-owned browser server command, when configured.
func (c Config) WebCommand() string {
	return c.Services.Web.Command
}

// UsesWebCommand reports whether dev should run a project-owned web server
// instead of wasmserve.
func (c Config) UsesWebCommand() bool {
	return c.Services.Web.Command != ""
}

// WebPort returns the port for the browser-facing web service.
func (c Config) WebPort() int {
	return c.Services.Web.Port
}

// WebPorts returns every configured browser-facing port.
func (c Config) WebPorts() []PortConfig {
	return append([]PortConfig(nil), c.Services.Web.Ports...)
}

// DashboardPort returns the local dashboard port.
func (c Config) DashboardPort() int {
	return c.Dashboard.Port
}

// APIEnabled reports whether the optional backend process should be started.
func (c Config) APIEnabled() bool {
	return c.Services.API.Enabled
}

// APICommand returns the shell-like command string for the optional backend.
func (c Config) APICommand() string {
	return c.Services.API.Command
}

// APIPort returns the configured backend port for display/status purposes.
func (c Config) APIPort() int {
	return c.Services.API.Port
}

// APIPorts returns every configured backend/API port.
func (c Config) APIPorts() []PortConfig {
	return append([]PortConfig(nil), c.Services.API.Ports...)
}

// NamedServices returns every configured service keyed by service name.
func (c Config) NamedServices() map[string]ServiceConfig {
	services := map[string]ServiceConfig{
		"web": c.Services.Web,
		"api": c.Services.API,
	}
	for name, service := range c.Services.Extra {
		services[name] = service
	}
	return services
}

// EnabledServices returns services that should be included in Docker Compose.
// The web service is always included because it is the browser entrypoint.
func (c Config) EnabledServices() map[string]ServiceConfig {
	services := map[string]ServiceConfig{
		"web": c.Services.Web,
	}
	if c.APIEnabled() {
		services["api"] = c.Services.API
	}
	for name, service := range c.Services.Extra {
		if service.Enabled {
			services[name] = service
		}
	}
	return services
}

// WASMExecPath returns where the matching Go wasm_exec.js should be copied.
func (c Config) WASMExecPath() string {
	return c.WASM.Exec
}

// BeforeRebuildCheckEnabled reports whether dev should gate rebuilds on a
// configured command.
func (c Config) BeforeRebuildCheckEnabled() bool {
	return c.Checks.BeforeRebuild.Enabled
}

// BeforeRebuildCheckCommand returns the command used before dev rebuilds.
func (c Config) BeforeRebuildCheckCommand() string {
	return c.Checks.BeforeRebuild.Command
}

// RebuildWatch returns patterns that should trigger a WASM rebuild.
func (c Config) RebuildWatch() []string {
	return append([]string(nil), c.Watch.Rebuild...)
}

// StaticWatch returns patterns that are part of the user-owned static web app.
func (c Config) StaticWatch() []string {
	return append([]string(nil), c.Watch.Static...)
}

// WatchPatterns returns every pattern dev mode should subscribe to.
func (c Config) WatchPatterns() []string {
	patterns := append([]string(nil), c.Watch.Rebuild...)
	patterns = append(patterns, c.Watch.Static...)
	return patterns
}
