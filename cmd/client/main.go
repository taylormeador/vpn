package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tmeador/vpn/internal/shared"
	"github.com/tmeador/vpn/internal/wg"
)

func main() {
	WG_ADDR := shared.MustEnv("WG_ADDR")
	SERVER_URL := shared.MustEnv("SERVER_URL")

	cfg, err := wg.LoadOrCreateConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	newPeers, err := registerWithServer(SERVER_URL, cfg.PublicKey, WG_ADDR)
	if err != nil {
		fmt.Printf("control plane unreachable, using cached peers: %v\n", err)
	} else {
		cfg.Peers = newPeers
		if err = cfg.WriteConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if err = wg.CreateAndConfigureInterface(*cfg, WG_ADDR); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("polling for peer updates every 30s")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		newPeers, err := registerWithServer(SERVER_URL, cfg.PublicKey, WG_ADDR)
		if err != nil {
			fmt.Printf("poll failed: %v\n", err)
			continue
		}

		cfg.Peers = newPeers
		if err = cfg.WriteConfig(); err != nil {
			fmt.Printf("failed to write config: %v\n", err)
			continue
		}

		if err = wg.ReconfigurePeers(*cfg); err != nil {
			fmt.Printf("failed to reconfigure peers: %v\n", err)
		}
	}
}
