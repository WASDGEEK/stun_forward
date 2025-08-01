// types.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PortMap defines a single port forwarding rule.
// The format for the string representation is "proto:local:remote".
type PortMap struct {
	Proto      string `json:"proto"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
}

// Config holds the application configuration, loaded from a JSON file.
type Config struct {
	Mode       string    `json:"mode"`
	Room       string    `json:"room"`
	SignalURL  string    `json:"signalURL"`
	StunServer string    `json:"stunServer,omitempty"`
	Mappings   []PortMap `json:"mappings"`
}

// UnmarshalJSON allows PortMap to be parsed from a simple string format.
func (pm *PortMap) UnmarshalJSON(data []byte) error {
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

	pm.Proto = proto
	pm.LocalPort = local
	pm.RemotePort = remote
	return nil
}
