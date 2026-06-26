package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"ebitdock/internal/config"
)

// ComposeFile is the subset of the compose spec that ebitdock writes. Keeping
// this narrow makes the generated file readable and predictable.
type ComposeFile struct {
	Name     string                    `yaml:"name,omitempty"`
	Services map[string]ComposeService `yaml:"services"`
	Volumes  map[string]struct{}       `yaml:"volumes,omitempty"`
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
	DependsOn   []string          `yaml:"depends_on,omitempty"`
}

// GenerateCompose builds the compose model from normalized ebitdock config.
func GenerateCompose(cfg config.Config) ComposeFile {
	return generateCompose("", cfg)
}

func generateCompose(root string, cfg config.Config) ComposeFile {
	services := map[string]ComposeService{}
	volumes := map[string]struct{}{}
	for name, service := range cfg.EnabledServices() {
		services[name] = composeService(root, service)
		for _, volume := range namedVolumes(service.Volumes) {
			volumes[volume] = struct{}{}
		}
	}
	return ComposeFile{
		Name:     cfg.Project,
		Services: services,
		Volumes:  volumes,
	}
}

// WriteCompose writes the generated compose file to the configured project path.
func WriteCompose(root string, cfg config.Config) (string, error) {
	path := filepath.Join(root, cfg.ComposeFile())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(generateCompose(root, cfg))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func composeService(root string, service config.ServiceConfig) ComposeService {
	out := ComposeService{
		Image:       service.Image,
		WorkingDir:  service.Workdir,
		Command:     service.Command,
		Environment: service.Env,
		Volumes:     normalizeBindVolumes(root, service.Volumes),
		Ports:       composePorts(service),
		DependsOn:   append([]string(nil), service.DependsOn...),
	}
	if service.Dockerfile != "" {
		out.Build = dockerfileBuildPath(service.Dockerfile)
		out.Image = ""
	}
	return out
}

func normalizeBindVolumes(root string, volumes []string) []string {
	out := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		if root == "" {
			out = append(out, volume)
			continue
		}
		host, rest, ok := strings.Cut(volume, ":")
		if !ok || host == "" || isNamedVolumeHost(host) {
			out = append(out, volume)
			continue
		}
		if !filepath.IsAbs(host) {
			host = filepath.Join(root, host)
		}
		out = append(out, filepath.Clean(host)+":"+rest)
	}
	return out
}

func dockerfileBuildPath(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return "."
	}
	if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
		return "./" + dir
	}
	return dir
}

func composePorts(service config.ServiceConfig) []string {
	out := make([]string, 0, len(service.Ports))
	for _, port := range service.Ports {
		if port.Port == 0 {
			continue
		}
		host := strconv.Itoa(port.Port)
		target := host
		if service.Kind == "static" {
			target = "80"
		}
		out = append(out, fmt.Sprintf("%s:%s", host, target))
	}
	return out
}

func namedVolumes(volumes []string) []string {
	var out []string
	for _, volume := range volumes {
		name, _, ok := strings.Cut(volume, ":")
		if !ok || name == "" {
			continue
		}
		if !isNamedVolumeHost(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func isNamedVolumeHost(host string) bool {
	return !strings.HasPrefix(host, ".") && !strings.HasPrefix(host, "/") && !strings.HasPrefix(host, "~")
}
