package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tmeador/vpn/internal/wg"
)

func registerWithServer(serverURL, pubkey, meshIP string) ([]wg.PeerEntry, error) {
	payload := map[string]string{
		"token":   "token-123",
		"pubkey":  pubkey,
		"mesh_ip": meshIP,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling join request: %w", err)
	}

	req, err := http.NewRequest("POST", serverURL+"/join", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating join request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contacting control plane: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("control plane returned %d", resp.StatusCode)
	}

	var peers []wg.PeerEntry
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("parsing peer list: %w", err)
	}

	return peers, nil
}
