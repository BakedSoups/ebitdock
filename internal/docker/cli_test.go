package docker

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCheckDockerUsesLookup(t *testing.T) {
	path, err := CheckDocker(func(name string) (string, error) {
		if name != "docker" {
			t.Fatalf("lookup name = %q, want docker", name)
		}
		return "/usr/bin/docker", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/usr/bin/docker" {
		t.Fatalf("path = %q", path)
	}
}

func TestRequireDockerIncludesInstallGuidance(t *testing.T) {
	err := RequireDocker(func(string) (string, error) {
		return "", errors.New("missing")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "compose plugin") {
		t.Fatalf("error should mention compose plugin: %v", err)
	}
}

func TestComposeCommand(t *testing.T) {
	name, args, err := ComposeCommand(".ebitdock/compose.yaml", "up", "--remove-orphans")
	if err != nil {
		t.Fatal(err)
	}
	if name != "docker" {
		t.Fatalf("name = %q, want docker", name)
	}
	want := []string{"compose", "-f", ".ebitdock/compose.yaml", "up", "--remove-orphans"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}
