package dashboard

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"ebitdock/internal/config"
	"ebitdock/internal/process"
)

//go:embed dashboard.html
var page string

// Run starts the dashboard and prints its URL.
func Run(ctx context.Context, root string, cfg config.Config, status *process.Status) error {
	return run(ctx, root, cfg, status, true)
}

// RunQuiet starts the dashboard without printing; dev mode owns terminal output.
func RunQuiet(ctx context.Context, root string, cfg config.Config, status *process.Status) error {
	return run(ctx, root, cfg, status, false)
}

// run serves both the HTML dashboard and the JSON status endpoint from the same
// in-memory Status object.
func run(ctx context.Context, root string, cfg config.Config, status *process.Status, printURL bool) error {
	_ = root
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl := template.Must(template.New("dashboard").Parse(page))
		_ = tpl.Execute(w, status.Snapshot())
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status.Snapshot())
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.DashboardPort()),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	status.AppendLog(fmt.Sprintf("dashboard listening on http://localhost:%d", cfg.DashboardPort()))
	if printURL {
		fmt.Printf("dashboard: http://localhost:%d\n", cfg.DashboardPort())
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
