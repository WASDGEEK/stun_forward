// run.go
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
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

	localAddr, err := getLocalAddress()
	if err != nil {
		log.Fatalf("could not get local address: %v", err)
	}

	// The peer port is the one we listen on for peer connections.
	// For the sender, this is an ephemeral port. For the receiver, it's specified.
	// This part of the logic is simplified and might need a proper STUN implementation.
	// For now, we use RemotePort for the receiver's peer port.
	peerPort := m.RemotePort
	myInfo := fmt.Sprintf("%s:%d", localAddr, peerPort)

	roomKey := cfg.Room + "-" + protoPortKey(m)
	err = PostSignal(cfg.SignalURL, cfg.Mode, roomKey, myInfo)
	if err != nil {
		log.Fatalf("signal post failed: %v", err)
	}

	peerInfo, err := WaitForPeerData(cfg.SignalURL, peer(cfg.Mode), roomKey, 30*time.Second)
	if err != nil {
		log.Fatalf("waiting for peer failed: %v", err)
	}

	host, portStr, ok := strings.Cut(peerInfo, ":")
	if !ok {
		log.Fatalf("invalid peer info format: %s", peerInfo)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid peer port from string '%s': %v", portStr, err)
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

func getLocalAddress() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return localAddr.IP.String(), nil
	}
	return "", errors.New("could not determine local IP address")
}