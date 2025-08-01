// types.go
package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type PortMap struct {
	Proto      string // "tcp" or "udp"
	LocalPort  int
	RemotePort int
}

type Config struct {
	Mode      string
	Room      string
	SignalURL string
	Mappings  []PortMap
}

func ParsePortMap(s string) (PortMap, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return PortMap{}, errors.New("port map must be in proto:local:remote format")
	}
	proto := strings.ToLower(parts[0])
	if proto != "tcp" && proto != "udp" {
		return PortMap{}, errors.New("protocol must be tcp or udp")
	}
	local, err1 := strconv.Atoi(parts[1])
	remote, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		return PortMap{}, fmt.Errorf("invalid port numbers in map: %v, %v", err1, err2)
	}
	return PortMap{Proto: proto, LocalPort: local, RemotePort: remote}, nil
}
