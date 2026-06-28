package install

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestToolsReportsMissingGo(t *testing.T) {
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})
	if err := os.Setenv("PATH", ""); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Tools(&out)
	if err == nil {
		t.Fatal("expected missing go error")
	}
	for _, want := range []string{"go executable not found", "ebitdock install tools"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q did not contain %q", err, want)
		}
	}
}
