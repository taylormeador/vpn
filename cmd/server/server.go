package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tmeador/vpn/internal/shared"
	"github.com/tmeador/vpn/internal/wg"
)
type Server struct {
	peers  *Peers
	logger *slog.Logger
	mux    *http.ServeMux
}

func NewServer(peers *Peers, logger *slog.Logger) *Server {
	s := &Server{
		peers:  peers,
		logger: logger,
		mux:    http.NewServeMux(),
	}
	s.mux.HandleFunc("/join", s.handleJoin)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(sw, r)
	s.logger.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", sw.status,
		"remote", r.RemoteAddr,
		"duration", time.Since(start),
	)
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	var req shared.JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	meshIP := strings.Split(req.MeshIP, "/")[0]

	newPeer := wg.PeerEntry{
		PublicKey:  req.PubKey,
		Endpoint:   remoteIP + ":51820",
		AllowedIPs: []string{meshIP + "/32"},
	}
	if err := s.peers.AddPeer(newPeer); err != nil {
		s.logger.Error("failed to add peer", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filtered := s.peers.GetPeers(req.PubKey)

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

