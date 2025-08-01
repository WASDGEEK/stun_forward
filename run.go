// Package main - Main runner for P2P port forwarding
package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"
)

// peerRole returns the opposite role for peer matching
func peerRole(mode string) string {
	if mode == "sender" {
		return "receiver"
	}
	return "sender"
}

// runForwarder starts the P2P port forwarding system
func runForwarder(config Configuration) {
	ctx := context.Background()
	
	for _, mapping := range config.Mappings {
		go handlePortMapping(ctx, config, mapping)
	}
	
	// Block forever
	select {}
}

// handlePortMapping handles a single port mapping configuration
func handlePortMapping(ctx context.Context, config Configuration, mapping PortMapping) {
	log.Printf("[%s] Starting port forward: %s %d <-> %d", 
		config.Mode, mapping.Protocol, mapping.LocalPort, mapping.RemotePort)

	// Discover public IP via STUN server
	log.Printf("Discovering public IP via STUN server: %s", config.STUNServer)
	publicAddr, err := getPublicIP(config.STUNServer, 5*time.Minute)
	if err != nil {
		log.Fatalf("Failed to get public IP: %v", err)
	}
	log.Printf("Discovered public address: %s", publicAddr)

	// Create signaling client
	signalingClient := NewSignalingClient()
	defer signalingClient.Close()

	// Generate unique room key for this mapping
	roomKey := config.RoomID + "-" + generateMappingKey(mapping)
	
	// Post our address to signaling server
	err = signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, publicAddr)
	if err != nil {
		log.Fatalf("Failed to post signal: %v", err)
	}

	// Wait for peer address
	peerAddr, err := signalingClient.WaitForPeerData(ctx, config.SignalingURL, 
		peerRole(config.Mode), roomKey, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to get peer address: %v", err)
	}

	// Parse peer address
	host, portStr, err := net.SplitHostPort(peerAddr)
	if err != nil {
		log.Fatalf("Invalid peer address format %s: %v", peerAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid peer port %s: %v", portStr, err)
	}

	log.Printf("Connected to peer at %s:%d", host, port)

	// Start appropriate forwarder based on mode and protocol
	if config.Mode == "sender" {
		if mapping.Protocol == "tcp" {
			runTCPSender(ctx, mapping.LocalPort, host, port)
		} else {
			runUDPSender(ctx, mapping.LocalPort, host, port)
		}
	} else {
		if mapping.Protocol == "tcp" {
			runTCPReceiver(ctx, mapping, host, port)
		} else {
			runUDPReceiver(ctx, mapping, host, port)
		}
	}
}

// generateMappingKey creates a unique key for the port mapping
func generateMappingKey(mapping PortMapping) string {
	// Sort ports to ensure consistent key regardless of sender/receiver
	local, remote := mapping.LocalPort, mapping.RemotePort
	if local > remote {
		local, remote = remote, local
	}
	return mapping.Protocol + "-" + strconv.Itoa(local) + "-" + strconv.Itoa(remote)
}