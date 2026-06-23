package cliui

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"ebitdock/internal/build"
	"ebitdock/internal/config"
)

func DevStatus(w io.Writer, cfg config.Config, result build.Result) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tSTATUS\tURL/DETAILS")
	fmt.Fprintf(tw, "web\trunning\thttp://localhost:%d\n", cfg.WebPort())
	fmt.Fprintf(tw, "dashboard\trunning\thttp://localhost:%d\n", cfg.DashboardPort())
	if cfg.APIEnabled() {
		fmt.Fprintf(tw, "backend\trunning\t:%d\n", cfg.APIPort())
	} else {
		fmt.Fprintln(tw, "backend\tdisabled\t-")
	}
	if result.Err != nil {
		fmt.Fprintf(tw, "wasm\tfailed\t%s\n", result.Err)
	} else {
		fmt.Fprintf(tw, "wasm\tok\t%s\n", roundDuration(result.Duration))
	}
	fmt.Fprintf(tw, "watch\tactive\t%d patterns\n", len(cfg.WatchPatterns()))
	_ = tw.Flush()
}

func BuildEvent(w io.Writer, result build.Result) {
	if result.Err != nil {
		fmt.Fprintf(w, "wasm\tfailed\t%s\n", result.Err)
		return
	}
	fmt.Fprintf(w, "wasm\tok\t%s\n", roundDuration(result.Duration))
}

func roundDuration(d time.Duration) time.Duration {
	if d > time.Second {
		return d.Round(10 * time.Millisecond)
	}
	return d.Round(time.Millisecond)
}
