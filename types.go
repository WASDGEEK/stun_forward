// types.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PortMapping defines a single port forwarding rule.
// The format for the string representation is "proto:local:remote".
type PortMapping struct {
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
}

// Configuration holds the application configuration, loaded from a JSON file.
type Configuration struct {
	Mode         string        `json:"mode"`
	RoomID       string        `json:"roomId"`
	SignalingURL string        `json:"signalingUrl"`
	STUNServer   string        `json:"stunServer,omitempty"`
	Mappings     []PortMapping `json:"mappings"`
}

// SignalingData represents data exchanged with signaling server
type SignalingData struct {
	Role string `json:"role"`
	Room string `json:"room"`
	Data string `json:"data"`
}

// UnmarshalJSON allows PortMapping to be parsed from a simple string format.
func (pm *PortMapping) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("port map must be a string: %w", err)
	}

	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return errors.New("port map must be in proto:local:remote format")
	}

	proto := strings.ToLower(parts[0])
	if proto != "tcp" && proto != "udp" {
		return errors.New("protocol must be tcp or udp")
	}

	local, err1 := strconv.Atoi(parts[1])
	remote, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid port numbers in map: %v, %v", err1, err2)
	}

	pm.Protocol = proto
	pm.LocalPort = local
	pm.RemotePort = remote
	return nil
}
