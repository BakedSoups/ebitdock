package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	apiURL := getenv("API_URL", "http://localhost:3001")
	realtimeURL := getenv("REALTIME_URL", "http://localhost:3002")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","service":"admin"}`))
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html>
<title>orbit-snake admin</title>
<body style="font-family:monospace;background:#090d12;color:#d9f7ff">
<h1>orbit-snake admin</h1>
<ul>
  <li>api: <a href="%[1]s/health">%[1]s</a></li>
  <li>realtime: <a href="%[2]s/health">%[2]s</a></li>
</ul>
<p>Debug controls can live here: reset arena, spawn crystals, inspect players.</p>
</body>`, apiURL, realtimeURL)
	})
	log.Printf("admin listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
