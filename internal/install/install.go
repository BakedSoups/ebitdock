package install

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/BakedSoups/ebitdock/internal/tools"
)

// Tools installs Go-based helper tools that ebitdock can manage directly.
// Docker is intentionally guidance-only because installation is OS-specific.
func Tools(w io.Writer) error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go executable not found in PATH\n\nInstall Go first, then rerun:\n  ebitdock install tools")
	}
	fmt.Fprintf(w, "installing wasmserve\n  %s\n", tools.WasmserveInstallCommand)
	cmd := exec.Command(goPath, "install", tools.WasmservePackage)
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install wasmserve failed: %w", err)
	}
	fmt.Fprintln(w, "wasmserve installed")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Docker is also required for default ebitdock dev.")
	fmt.Fprintln(w, "Install Docker Desktop or Docker Engine with the Compose plugin enabled.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Next:")
	fmt.Fprintln(w, "  ebitdock doctor")
	return nil
}
