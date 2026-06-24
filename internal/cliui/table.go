package cliui

import (
	"fmt"
	"io"
	"text/tabwriter"

	"ebitdock/internal/config"
)

// DevStatus prints the startup service table. tabwriter keeps the output
// Docker-like without adding a formatting dependency.
func DevStatus(w io.Writer, cfg config.Config) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tSTATUS\tURL/DETAILS")
	fmt.Fprintf(tw, "wasmserve\trunning\thttp://localhost:%d\n", cfg.WebPort())
	fmt.Fprintf(tw, "dashboard\trunning\thttp://localhost:%d\n", cfg.DashboardPort())
	if cfg.APIEnabled() {
		fmt.Fprintf(tw, "backend\trunning\t:%d\n", cfg.APIPort())
	} else {
		fmt.Fprintln(tw, "backend\tdisabled\t-")
	}
	fmt.Fprintf(tw, "watch\tactive\t%d patterns\n", len(cfg.WatchPatterns()))
	_ = tw.Flush()
}
