package wg

import (
	"errors"
	"fmt"
	"syscall"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/vishvananda/netlink"
)

func ReconfigurePeers(cfg LocalWGConfig) error {
	client, err := wgctrl.New()
	if err != nil {
		return err
	}
	defer client.Close()

	wgtypesCfg, err := cfg.ToWGTypes()
	if err != nil {
		return err
	}

	return client.ConfigureDevice("wg0", wgtypes.Config{
		ReplacePeers: true,
		Peers:        wgtypesCfg.Peers,
	})
}

func CreateAndConfigureInterface(cfg LocalWGConfig, wgAddr string) error {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = "wg0"

	link := &netlink.GenericLink{
		LinkAttrs: attrs,
		LinkType:  "wireguard",
	}

	if err := netlink.LinkAdd(link); err != nil {
		if errors.Is(err, syscall.EEXIST) {
			fmt.Println("interface already exists.")
		} else {
			return err
		}
	} else {
		fmt.Println("WireGuard interface wg0 created successfully.")
	}

	addr, err := netlink.ParseAddr(wgAddr)
	if err != nil {
		return err
	}
	if err = netlink.AddrReplace(link, addr); err != nil {
		return err
	}
	fmt.Printf("assigned wg0 ip address %s\n", addr)

	if err = netlink.LinkSetUp(link); err != nil {
		return err
	}
	fmt.Println("interface up")

	client, err := wgctrl.New()
	if err != nil {
		return err
	}
	defer client.Close()

	wgtypesCfg, err := cfg.ToWGTypes()
	if err != nil {
		return err
	}
	
	if err = client.ConfigureDevice("wg0", *wgtypesCfg); err != nil {
		return err
	}
	fmt.Println("wireguard configured")

	return nil
}
