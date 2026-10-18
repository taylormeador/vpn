package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/vishvananda/netlink"
)

const (
	configDir  = "/etc/vpn"
	configPath = "/etc/vpn/config.json"
)

func main() {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = "wg0"

	link := &netlink.GenericLink{
		LinkAttrs: attrs,
		LinkType:  "wireguard",
	}

	if err := netlink.LinkAdd(link); err != nil {
		if errors.Is(err, syscall.EEXIST) {
			fmt.Println("Interface already exists.")
		} else {
			fmt.Printf("failed to create wireguard interface: %v\n", err)
			return
		}
	} else {
		fmt.Println("WireGuard interface wg0 created successfully.")
	}

	WG_ADDR := os.Getenv("WG_ADDR")
	addr, err := netlink.ParseAddr(WG_ADDR)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = netlink.AddrAdd(link, addr); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("assigned wg0 ip address %s\n", WG_ADDR)

	if err = netlink.LinkSetUp(link); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("interface up")

	client, err := wgctrl.New()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close()

	cfg, err := loadOrCreateConfig()
	if err != nil {
		fmt.Printf("config error: %v\n", err)
		return
	}

	if err = client.ConfigureDevice("wg0", *cfg); err != nil {
		fmt.Printf("failed to configure device: %v\n", err)
		return
	}
	fmt.Println("wireguard configured")
}

type PeerEntry struct {
	PublicKey  string `json:"public_key"`
	Endpoint  string `json:"endpoint"`
	AllowedIPs []string `json:"allowed_ips"`
}

type WireguardConfig struct {
	PrivateKey string      `json:"private_key"`
	ListenPort int         `json:"listen_port"`
	Peer       *PeerEntry  `json:"peer,omitempty"`
}

func generateConfig() (*WireguardConfig, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}

	cfg := &WireguardConfig{
		PrivateKey: privateKey.String(),
		ListenPort: 51820,
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return nil, fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("generated new config at %s\n", configPath)
	return cfg, nil
}

func loadOrCreateConfig() (*wgtypes.Config, error) {
	var wgConfig *WireguardConfig

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
		wgConfig = &WireguardConfig{}
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

	var peers []wgtypes.PeerConfig
	if wgConfig.Peer != nil {
		peerKey, err := wgtypes.ParseKey(wgConfig.Peer.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("parsing peer public key: %w", err)
		}

		endpoint, err := net.ResolveUDPAddr("udp", wgConfig.Peer.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("resolving peer endpoint: %w", err)
		}

		var allowedIPs []net.IPNet
		for _, cidr := range wgConfig.Peer.AllowedIPs {
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

	return &wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &wgConfig.ListenPort,
		ReplacePeers: true,
		Peers:        peers,
	}, nil
}
