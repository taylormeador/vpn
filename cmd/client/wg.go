package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/tmeador/vpn/internal/types"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type LocalWGConfig struct {
	PrivateKey string            `json:"private_key"`
	ListenPort int               `json:"listen_port"`
	Peers      []types.PeerEntry `json:"peers"`
}

func writeConfig(cfg *LocalWGConfig) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func generateConfig() (*LocalWGConfig, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}

	cfg := &LocalWGConfig{
		PrivateKey: privateKey.String(),
		ListenPort: 51820,
	}

	if err = writeConfig(cfg); err != nil {
		return nil, fmt.Errorf("writing new config:  %w", err)
	}

	fmt.Printf("generated new config at %s\n", configPath)
	return cfg, nil
}

func loadOrCreateConfig() (*wgtypes.Config, error) {

	// Try to read the local config, fallback to generating a new one
	var wgConfig *LocalWGConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			wgConfig, err = generateConfig()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	} else {
		wgConfig = &LocalWGConfig{}
		if err := json.Unmarshal(data, wgConfig); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		fmt.Printf("loaded config from %s\n", configPath)
	}

	privateKey, err := wgtypes.ParseKey(wgConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}
	fmt.Printf("public key: %s\n", privateKey.PublicKey())

	// Get updated peer list from control plane.
	// Fallback to pre-existing local peer list in case of server failure.
	pubkey := privateKey.PublicKey().String()
	serverURL := "http://10.0.10.110:8080"
	endpoint := os.Getenv("WG_ADDR")
	newPeers, err := registerWithServer(serverURL, pubkey, endpoint)
	if err != nil {
		fmt.Printf("control plane unreachable, using cached peers: %v\n", err)
	} else {
		wgConfig.Peers = newPeers
		if err = writeConfig(wgConfig); err != nil {
			return nil, fmt.Errorf("writing config: %w", err)
		}
	}

	peerConfigs, err := getPeerConfigs(wgConfig.Peers)
	if err != nil {
		return nil, fmt.Errorf("parsing peer configs: %w", err)
	}

	return &wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &wgConfig.ListenPort,
		ReplacePeers: true,
		Peers:        *peerConfigs,
	}, nil
}

func getPeerConfigs(peerEntries []types.PeerEntry) (*[]wgtypes.PeerConfig, error) {
	var peers []wgtypes.PeerConfig

	for _, entry := range peerEntries {
		peerKey, err := wgtypes.ParseKey(entry.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("parsing peer public key: %w", err)
		}

		endpoint, err := net.ResolveUDPAddr("udp", entry.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("resolving peer endpoint: %w", err)
		}

		var allowedIPs []net.IPNet
		for _, cidr := range entry.AllowedIPs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, fmt.Errorf("parsing allowed IP %s: %w", cidr, err)
			}
			allowedIPs = append(allowedIPs, *ipNet)
		}

		peers = append(peers, wgtypes.PeerConfig{
			PublicKey:  peerKey,
			Endpoint:   endpoint,
			AllowedIPs: allowedIPs,
		})
	}

	return &peers, nil
}
