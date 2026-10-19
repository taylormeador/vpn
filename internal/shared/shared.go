package shared

import (
	"fmt"
	"os"
)

type JoinRequest struct {
	Token  string `json:"token"`
	PubKey string `json:"pubkey"`
	MeshIP string `json:"mesh_ip"`
}

func MustEnv(envVar string) string {
	val := os.Getenv(envVar)
	if val == "" {
		fmt.Println("env var must be set", "env var", envVar)
		os.Exit(1)
	}
	return val
}
