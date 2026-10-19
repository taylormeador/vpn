package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tmeador/vpn/internal/types"
)

var peers = []types.PeerEntry{
	{
		PublicKey:  "hyMPR8wC6QeKC+qjXYo8OlZV5Nu4Nnd958vRHnqEQEY=",
		Endpoint:   "10.0.10.110:51820",
		AllowedIPs: []string{"10.100.0.110/32"},
	},
	{
		PublicKey:  "LEQNlM54CfckGqkEGfBQij1s1t+3/D4C6cBFAlUToi4=",
		Endpoint:   "10.0.10.111:51820",
		AllowedIPs: []string{"10.100.0.111/32"},
	},
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	var req types.JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Return all peers except the requesting node
	var filtered []types.PeerEntry
	for _, p := range peers {
		if p.PublicKey != req.PubKey {
			filtered = append(filtered, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"remote", r.RemoteAddr,
			"duration", time.Since(start),
		)
	})
}

func mustEnv(envVar string) string {
	val := os.Getenv(envVar)
	if val == "" {
		fmt.Println("env var must be set", "env var", envVar)
		os.Exit(1)
	}
	return val
}

func main() {
	LISTEN_PORT := mustEnv("LISTEN_PORT")

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("starting control plane")

	mux := http.NewServeMux()
	mux.HandleFunc("/join", joinHandler)

	logger.Info("listening", "port", LISTEN_PORT)
	port := fmt.Sprintf(":%s", LISTEN_PORT)
	err := http.ListenAndServe(port, logMiddleware(logger, mux))
	if err != nil {
		logger.Error("Error starting server", "err", err)
	}
}
