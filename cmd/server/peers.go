package main

import (
	"sync"

	"github.com/tmeador/vpn/internal/wg"
)

type Peers struct {
	mu      sync.RWMutex
	entries map[string]wg.PeerEntry
	cfg     *wg.LocalWGConfig
}

func NewPeers(cfg *wg.LocalWGConfig) *Peers {
	entries := make(map[string]wg.PeerEntry)
	for _, p := range cfg.Peers {
		entries[p.PublicKey] = p
	}
	return &Peers{
		entries: entries,
		cfg:     cfg,
	}
}

func (p *Peers) AddPeer(peer wg.PeerEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.entries[peer.PublicKey] = peer

	peers := make([]wg.PeerEntry, 0, len(p.entries))
	for _, entry := range p.entries {
		peers = append(peers, entry)
	}
	p.cfg.Peers = peers

	if err := p.cfg.WriteConfig(); err != nil {
		return err
	}

	return wg.ReconfigurePeers(*p.cfg)
}

func (p *Peers) GetPeers(nodePubKey string) []wg.PeerEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var filtered []wg.PeerEntry
	for pubKey, entry := range p.entries {
		if pubKey == nodePubKey {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
