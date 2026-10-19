package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/tmeador/vpn/internal/shared"
	"github.com/tmeador/vpn/internal/wg"
)

func main() {
	LISTEN_PORT := shared.MustEnv("LISTEN_PORT")
	WG_ADDR := shared.MustEnv("WG_ADDR")
	LAN_IP := shared.MustEnv("LAN_IP")

	cfg, err := wg.LoadOrCreateConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = wg.CreateAndConfigureInterface(*cfg, WG_ADDR); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	meshIP := strings.Split(WG_ADDR, "/")[0]
	peers := NewPeers(cfg)
	peers.AddPeer(wg.PeerEntry{
		PublicKey:  cfg.PublicKey,
		Endpoint:   LAN_IP + ":51820",
		AllowedIPs: []string{meshIP + "/32"},
	})

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("starting control plane")

	srv := NewServer(peers, logger)

	logger.Info("listening", "port", LISTEN_PORT)
	port := fmt.Sprintf(":%s", LISTEN_PORT)
	if err = http.ListenAndServe(port, srv); err != nil {
		logger.Error("error starting server", "err", err)
	}
}
