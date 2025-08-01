// run.go
package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func peer(mode string) string {
	if mode == "sender" {
		return "receiver"
	}
	return "sender"
}

func Run(cfg Config) {
	for _, m := range cfg.Mappings {
		go handleMapping(cfg, m)
	}
	select {} // block forever
}

func handleMapping(cfg Config, m PortMap) {
	log.Printf("[%s] Preparing port forward: %s %d <-> %d", cfg.Mode, m.Proto, m.LocalPort, m.RemotePort)

	log.Printf("Discovering public IP via STUN server: %s", cfg.StunServer)
	publicAddr, err := getPublicIP(cfg.StunServer)
	if err != nil {
		log.Fatalf("Failed to get public IP: %v", err)
	}
	log.Printf("Discovered public address: %s", publicAddr)

	roomKey := cfg.Room + "-" + protoPortKey(m)
	err = PostSignal(cfg.SignalURL, cfg.Mode, roomKey, publicAddr)
	if err != nil {
		log.Fatalf("Signal post failed: %v", err)
	}

	peerInfo, err := WaitForPeerData(cfg.SignalURL, peer(cfg.Mode), roomKey, 30*time.Second)
	if err != nil {
		log.Fatalf("Waiting for peer failed: %v", err)
	}

	host, portStr, ok := strings.Cut(peerInfo, ":")
	if !ok {
		log.Fatalf("Invalid peer info format: %s", peerInfo)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid peer port from string '%s': %v", portStr, err)
	}

	if cfg.Mode == "sender" {
		// sender connects to peer, receives from local
		if m.Proto == "tcp" {
			tcpSender(m.LocalPort, host, port)
		} else {
			udpSender(m.LocalPort, host, port)
		}
	} else {
		// receiver listens on its peer port, connects to local service
		if m.Proto == "tcp" {
			tcpReceiver(m.LocalPort, host, port)
		} else {
			udpReceiver(m.LocalPort, host, port)
		}
	}
}

func protoPortKey(m PortMap) string {
	// Sort ports to ensure the key is the same regardless of order
	if m.LocalPort > m.RemotePort {
		return fmt.Sprintf("%s-%d-%d", m.Proto, m.RemotePort, m.LocalPort)
	}
	return fmt.Sprintf("%s-%d-%d", m.Proto, m.LocalPort, m.RemotePort)
}
