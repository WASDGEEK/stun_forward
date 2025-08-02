package types

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Mode represents the operation mode
type Mode string

const (
	ModeClient Mode = "client"
	ModeServer Mode = "server"
)

// PortMapping represents a port forwarding rule
type PortMapping struct {
	Protocol   string `json:"protocol" yaml:"protocol"`
	LocalPort  int    `json:"localPort" yaml:"localPort"`
	RemotePort int    `json:"remotePort" yaml:"remotePort"`
}

// ParsePortMapping parses a port mapping from string format "protocol:localPort:remotePort"
func ParsePortMapping(s string) (*PortMapping, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid mapping format, expected 'protocol:localPort:remotePort', got '%s'", s)
	}

	protocol := strings.ToLower(parts[0])
	if protocol != "tcp" && protocol != "udp" {
		return nil, fmt.Errorf("unsupported protocol '%s', must be 'tcp' or 'udp'", protocol)
	}

	localPort, err := strconv.Atoi(parts[1])
	if err != nil || localPort <= 0 || localPort > 65535 {
		return nil, fmt.Errorf("invalid local port '%s'", parts[1])
	}

	remotePort, err := strconv.Atoi(parts[2])
	if err != nil || remotePort <= 0 || remotePort > 65535 {
		return nil, fmt.Errorf("invalid remote port '%s'", parts[2])
	}

	return &PortMapping{
		Protocol:   protocol,
		LocalPort:  localPort,
		RemotePort: remotePort,
	}, nil
}

// String returns the string representation of the port mapping
func (pm *PortMapping) String() string {
	return fmt.Sprintf("%s:%d:%d", pm.Protocol, pm.LocalPort, pm.RemotePort)
}

// Config represents the application configuration
type Config struct {
	Mode         Mode           `json:"mode" yaml:"mode"`
	RoomID       string         `json:"roomId" yaml:"roomId"`
	SignalingURL string         `json:"signalingUrl" yaml:"signalingUrl"`
	STUNServer   string         `json:"stunServer" yaml:"stunServer"`
	Mappings     []*PortMapping `json:"mappings" yaml:"mappings"`
	
	// Advanced options
	ConnectTimeout time.Duration `json:"connectTimeout" yaml:"connectTimeout"`
	RetryCount     int           `json:"retryCount" yaml:"retryCount"`
	LogLevel       string        `json:"logLevel" yaml:"logLevel"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		STUNServer:     "stun.l.google.com:19302",
		ConnectTimeout: 30 * time.Second,
		RetryCount:     3,
		LogLevel:       "info",
		Mappings:       make([]*PortMapping, 0),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Mode != ModeClient && c.Mode != ModeServer {
		return fmt.Errorf("invalid mode '%s', must be 'client' or 'server'", c.Mode)
	}

	if c.RoomID == "" {
		return fmt.Errorf("roomId cannot be empty")
	}

	if c.SignalingURL == "" {
		return fmt.Errorf("signalingUrl cannot be empty")
	}

	if c.STUNServer == "" {
		return fmt.Errorf("stunServer cannot be empty")
	}

	// Validate STUN server format
	if _, _, err := net.SplitHostPort(c.STUNServer); err != nil {
		return fmt.Errorf("invalid stunServer format '%s': %w", c.STUNServer, err)
	}

	// Client mode requires mappings
	if c.Mode == ModeClient && len(c.Mappings) == 0 {
		return fmt.Errorf("client mode requires at least one port mapping")
	}

	// Validate port mappings
	for i, mapping := range c.Mappings {
		if mapping == nil {
			return fmt.Errorf("mapping %d is nil", i)
		}
		if mapping.Protocol != "tcp" && mapping.Protocol != "udp" {
			return fmt.Errorf("mapping %d: invalid protocol '%s'", i, mapping.Protocol)
		}
		if mapping.LocalPort <= 0 || mapping.LocalPort > 65535 {
			return fmt.Errorf("mapping %d: invalid local port %d", i, mapping.LocalPort)
		}
		if mapping.RemotePort <= 0 || mapping.RemotePort > 65535 {
			return fmt.Errorf("mapping %d: invalid remote port %d", i, mapping.RemotePort)
		}
	}

	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connectTimeout must be positive")
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("retryCount cannot be negative")
	}

	return nil
}