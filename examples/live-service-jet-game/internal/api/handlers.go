package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{store: store}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "service": "api"})
	})
	mux.HandleFunc("GET /players/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, s.store.Player(r.PathValue("id")))
	})
	mux.HandleFunc("POST /players/{id}/scrap", func(w http.ResponseWriter, r *http.Request) {
		amount, _ := strconv.Atoi(r.URL.Query().Get("amount"))
		if amount == 0 {
			amount = 1
		}
		writeJSON(w, s.store.AddScrap(r.PathValue("id"), amount))
	})
	mux.HandleFunc("GET /leaderboard", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, s.store.Leaderboard())
	})
	return cors(mux)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
