package wg

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	configDir  = "/etc/vpn"
	configPath = "/etc/vpn/config.json"
)

type PeerEntry struct {
	PublicKey  string   `json:"public_key"`
	Endpoint  string    `json:"endpoint"`
	AllowedIPs []string `json:"allowed_ips"`
}

type LocalWGConfig struct {
	PrivateKey string            `json:"private_key"`
	privateKey wgtypes.Key       `json:"-"`
	PublicKey  string            `json:"-"`
	ListenPort int               `json:"listen_port"`
	Peers      []PeerEntry       `json:"peers"`
}

func (c *LocalWGConfig) ToWGTypes() (*wgtypes.Config, error) {
	peers, err := c.parsePeerConfigs()
	if err != nil {
		return nil, err
	}
	
	cfg := &wgtypes.Config{
		PrivateKey: &c.privateKey,
		ListenPort: &c.ListenPort,
		ReplacePeers: true,
		Peers: *peers,
	}
	
	return cfg, nil
}

func (c *LocalWGConfig) parsePeerConfigs() (*[]wgtypes.PeerConfig, error) {
	var peers []wgtypes.PeerConfig
	for _, entry := range c.Peers {
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

func (c *LocalWGConfig) WriteConfig() error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
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
		privateKey: privateKey,
		PublicKey:  privateKey.PublicKey().String(),
		ListenPort: 51820,
	}

	if err = cfg.WriteConfig(); err != nil {
		return nil, fmt.Errorf("writing new config:  %w", err)
	}

	fmt.Printf("generated new config at %s\n", configPath)
	return cfg, nil
}

func LoadOrCreateConfig() (*LocalWGConfig, error) {

	// Try to read the local config, fallback to generating a new one
	var config *LocalWGConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			config, err = generateConfig()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	} else {
		config = &LocalWGConfig{}
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}

		key, err := wgtypes.ParseKey(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("parsing private key: %w", err)
		}
		config.privateKey = key
		config.PublicKey = key.PublicKey().String()
		fmt.Printf("loaded config from %s\n", configPath)
	}

	return config, nil
}