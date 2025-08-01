// types.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PortMapping defines a single port forwarding rule.
// The format for the string representation is "proto:local:remote".
type PortMapping struct {
	Protocol   string `json:"protocol" yaml:"protocol"`
	LocalPort  int    `json:"localPort" yaml:"localPort"`
	RemotePort int    `json:"remotePort" yaml:"remotePort"`
}

// Configuration holds the application configuration.
type Configuration struct {
	Mode         string        `json:"mode" yaml:"mode"`
	RoomID       string        `json:"roomId" yaml:"roomId"`
	SignalingURL string        `json:"signalingUrl" yaml:"signalingUrl"`
	STUNServer   string        `json:"stunServer,omitempty" yaml:"stunServer,omitempty"`
	Mappings     []PortMapping `json:"mappings,omitempty" yaml:"mappings,omitempty"`
}

// SignalingData represents data exchanged with signaling server
type SignalingData struct {
	Role string `json:"role"`
	Room string `json:"room"`
	Data string `json:"data"`
}

// NetworkInfo contains network connection information
type NetworkInfo struct {
	PublicAddr  string
	PrivateAddr string
	IsLAN       bool
}

// UnmarshalJSON allows PortMapping to be parsed from a simple string format.
func (pm *PortMapping) UnmarshalJSON(data []byte) error {
	return pm.unmarshalString(data, json.Unmarshal)
}

// UnmarshalYAML allows PortMapping to be parsed from a simple string format in YAML.
func (pm *PortMapping) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("port map must be a string: %w", err)
	}
	return pm.parseFromString(s)
}

// unmarshalString is a helper for both JSON and YAML parsing
func (pm *PortMapping) unmarshalString(data []byte, unmarshal func([]byte, interface{}) error) error {
	var s string
	if err := unmarshal(data, &s); err != nil {
		return fmt.Errorf("port map must be a string: %w", err)
	}
	return pm.parseFromString(s)
}

// parseFromString parses the port mapping from string format
func (pm *PortMapping) parseFromString(s string) error {
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
