package serve

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"ebitdock/internal/config"
	"ebitdock/internal/process"
)

func Run(ctx context.Context, root string, cfg config.Config, status *process.Status) error {
	return run(ctx, root, cfg, status, true)
}

func RunQuiet(ctx context.Context, root string, cfg config.Config, status *process.Status) error {
	return run(ctx, root, cfg, status, false)
}

func run(ctx context.Context, root string, cfg config.Config, status *process.Status, printURL bool) error {
	return runHandler(ctx, Server(root, cfg), cfg, status, printURL)
}

func runHandler(ctx context.Context, handler http.Handler, cfg config.Config, status *process.Status, printURL bool) error {
	addr := fmt.Sprintf(":%d", cfg.WebPort())
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	status.AppendLog(fmt.Sprintf("web server listening on http://localhost:%d", cfg.WebPort()))
	if printURL {
		fmt.Printf("web: http://localhost:%d\n", cfg.WebPort())
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

func Server(root string, cfg config.Config) http.Handler {
	_ = mime.AddExtensionType(".wasm", "application/wasm")
	webRoot := filepath.Join(root, cfg.StaticRoot())
	fileServer := http.FileServer(http.Dir(webRoot))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		fileServer.ServeHTTP(w, r)
	})
}
