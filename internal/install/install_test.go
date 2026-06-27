package install

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestToolsReportsMissingDocker(t *testing.T) {
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})
	if err := os.Setenv("PATH", ""); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Tools(&out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"docker not found", "Docker Desktop", "ebitdock doctor"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output %q did not contain %q", out.String(), want)
		}
	}
}
