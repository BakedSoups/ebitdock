package install

import (
	"fmt"
	"io"
	"os/exec"
)

// Tools installs Go-based helper tools that ebitdock can manage directly.
// Docker is intentionally guidance-only because installation is OS-specific.
func Tools(w io.Writer) error {
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(w, "docker not found in PATH")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Install Docker Desktop or Docker Engine with the Compose plugin enabled.")
		fmt.Fprintln(w, "Then rerun:")
		fmt.Fprintln(w, "  ebitdock doctor")
		return nil
	}
	fmt.Fprintln(w, "docker found")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "No additional Go helper tools are required for Docker-backed ebitdock dev.")
	fmt.Fprintln(w, "Docker installation is OS-specific, so ebitdock does not install it directly.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Next:")
	fmt.Fprintln(w, "  ebitdock doctor")
	return nil
}
