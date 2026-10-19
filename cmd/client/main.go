package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.zx2c4.com/wireguard/wgctrl"

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
