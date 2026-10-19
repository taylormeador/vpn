package types

type PeerEntry struct {
	PublicKey  string   `json:"public_key"`
	Endpoint  string   `json:"endpoint"`
	AllowedIPs []string `json:"allowed_ips"`
}

type JoinRequest struct {
	Token    string `json:"token"`
	PubKey   string `json:"pubkey"`
	Endpoint string `json:"endpoint"`
}
