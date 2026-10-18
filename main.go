package main

import (
	// "golang.zx2c4.com/wireguard/wgctrl"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/vishvananda/netlink"
)

func main() {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = "wg0"

	// Create a generic link with type "wireguard"
	link := &netlink.GenericLink{
		LinkAttrs: attrs,
		LinkType:  "wireguard",
	}

	// Add the link via netlink
	if err := netlink.LinkAdd(link); err != nil {
		// If the interface already exists, handle it idempotently
		if errors.Is(err, syscall.EEXIST) {
			fmt.Println("Interface already exists.")
		} else {
			// Return any other underlying error (e.g., permission denied)
			fmt.Printf("failed to create wireguard interface: %v", err)
			return
		}
	} else {
		fmt.Println("WireGuard interface wg0 created successfully.")
	}

	// Assign IP address
	WG_ADDR := os.Getenv("WG_ADDR")
	addr, err := netlink.ParseAddr(WG_ADDR)
	if err != nil {
		fmt.Println(err)
	}
	if err = netlink.AddrAdd(link, addr); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("assigned wg0 ip address %s\n", WG_ADDR)

	// bring the iface up
	if err = netlink.LinkSetUp(link); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("interface up")
	}
}
