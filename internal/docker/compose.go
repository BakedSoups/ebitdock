package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"

	"ebitdock/internal/config"
)

// ComposeFile is the subset of the compose spec that ebitdock writes. Keeping
// this narrow makes the generated file readable and predictable.
type ComposeFile struct {
	Name     string                    `yaml:"name,omitempty"`
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents one project service container.
type ComposeService struct {
	Image       string            `yaml:"image,omitempty"`
	Build       string            `yaml:"build,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
}

// GenerateCompose builds the compose model from normalized ebitdock config.
func GenerateCompose(cfg config.Config) ComposeFile {
	services := map[string]ComposeService{
		"web": composeService(cfg.Services.Web, cfg.WebPorts()),
	}
	if cfg.APIEnabled() {
		services["api"] = composeService(cfg.Services.API, cfg.APIPorts())
	}
	return ComposeFile{
		Name:     cfg.Project,
		Services: services,
	}
}

// WriteCompose writes the generated compose file to the configured project path.
func WriteCompose(root string, cfg config.Config) (string, error) {
	path := filepath.Join(root, cfg.ComposeFile())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(GenerateCompose(cfg))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func composeService(service config.ServiceConfig, ports []config.PortConfig) ComposeService {
	out := ComposeService{
		Image:       service.Image,
		WorkingDir:  service.Workdir,
		Command:     service.Command,
		Environment: service.Env,
		Volumes:     append([]string(nil), service.Volumes...),
		Ports:       composePorts(ports),
	}
	if service.Dockerfile != "" {
		out.Build = "."
		out.Image = ""
	}
	return out
}

func composePorts(ports []config.PortConfig) []string {
	out := make([]string, 0, len(ports))
	for _, port := range ports {
		if port.Port == 0 {
			continue
		}
		value := strconv.Itoa(port.Port)
		out = append(out, fmt.Sprintf("%s:%s", value, value))
	}
	return out
}
