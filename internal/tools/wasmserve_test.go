package tools

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestWasmserveCommand(t *testing.T) {
	name, args, err := WasmserveCommand(8081, "./cmd/game")
	if err != nil {
		t.Fatal(err)
	}
	if name != "wasmserve" {
		t.Fatalf("name = %q, want wasmserve", name)
	}
	want := []string{"-http", ":8081", "./cmd/game"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestWasmserveCommandRequiresGamePackage(t *testing.T) {
	_, _, err := WasmserveCommand(8081, "")
	if err == nil || !strings.Contains(err.Error(), "game.package is required") {
		t.Fatalf("err = %v, want game.package error", err)
	}
}

func TestWasmserveCommandRequiresPort(t *testing.T) {
	_, _, err := WasmserveCommand(0, "./cmd/game")
	if err == nil || !strings.Contains(err.Error(), "services.web.port is required") {
		t.Fatalf("err = %v, want services.web.port error", err)
	}
}

func TestCheckWasmserveMissingBinaryIncludesInstallCommand(t *testing.T) {
	err := CheckWasmserve(func(string) (string, error) {
		return "", errors.New("not found")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "wasmserve not found") {
		t.Fatalf("error did not mention missing wasmserve: %q", msg)
	}
	if !strings.Contains(msg, WasmserveInstallCommand) {
		t.Fatalf("error did not include install command: %q", msg)
	}
}

func TestCheckWasmserveFound(t *testing.T) {
	err := CheckWasmserve(func(name string) (string, error) {
		if name != "wasmserve" {
			t.Fatalf("lookup name = %q, want wasmserve", name)
		}
		return "/tmp/wasmserve", nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
