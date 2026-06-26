package docker

import (
	"errors"
	"fmt"
	"os/exec"
)

// LookupFunc lets tests replace exec.LookPath without changing PATH.
type LookupFunc func(string) (string, error)

// CheckDocker verifies that the docker CLI is installed.
func CheckDocker(lookup LookupFunc) (string, error) {
	if lookup == nil {
		lookup = exec.LookPath
	}
	path, err := lookup("docker")
	if err != nil {
		return "", errors.New("docker executable not found in PATH")
	}
	return path, nil
}

// ComposeArgs returns a docker compose argv using ebitdock's managed compose file.
func ComposeArgs(composeFile string, args ...string) ([]string, error) {
	if composeFile == "" {
		return nil, errors.New("docker.compose_file is required")
	}
	out := []string{"compose", "-f", composeFile}
	out = append(out, args...)
	return out, nil
}

// ComposeCommand returns the executable and argv for a docker compose command.
func ComposeCommand(composeFile string, args ...string) (string, []string, error) {
	composeArgs, err := ComposeArgs(composeFile, args...)
	if err != nil {
		return "", nil, err
	}
	return "docker", composeArgs, nil
}

// RequireDocker wraps CheckDocker with user-facing install guidance.
func RequireDocker(lookup LookupFunc) error {
	if _, err := CheckDocker(lookup); err != nil {
		return fmt.Errorf("%w\n\nInstall Docker Desktop or Docker Engine with the compose plugin enabled", err)
	}
	return nil
}
