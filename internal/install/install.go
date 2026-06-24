package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"text/tabwriter"

	"ebitdock/internal/tools"
)

// RunTools installs the external Go tools used by ebitdock dev.
func RunTools(ctx context.Context, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TOOL\tACTION\tDETAILS")
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(tw, "go\tmissing\tinstall Go and make sure go is on PATH")
		_ = tw.Flush()
		return errors.New("go executable not found in PATH")
	}

	fmt.Fprintf(tw, "wasmserve\tinstalling\t%s\n", tools.WasmservePackage)
	_ = tw.Flush()

	cmd := exec.CommandContext(ctx, "go", "install", tools.WasmservePackage)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install wasmserve: %w", err)
	}

	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "wasmserve\tok\tinstalled")
	return tw.Flush()
}
