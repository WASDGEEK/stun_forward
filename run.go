// Package main - Main runner for P2P port forwarding
package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// peerRole returns the opposite role for peer matching
func peerRole(mode string) string {
	if mode == "client" {
		return "server"
	}
	return "client"
}

// runForwarder starts the P2P port forwarding system
func runForwarder(config Configuration) {
	ctx := context.Background()
	
	if config.Mode == "client" {
		// Client mode: handle each mapping
		for _, mapping := range config.Mappings {
			go handlePortMapping(ctx, config, mapping)
		}
	} else {
		// Server mode: just wait for connections without specific mappings
		go handleServerMode(ctx, config)
	}
	
	// Block forever
	select {}
}

// handlePortMapping handles a single port mapping configuration (client mode)
func handlePortMapping(ctx context.Context, config Configuration, mapping PortMapping) {
	log.Printf("[%s] Starting port forward: %s %d -> %d", 
		config.Mode, mapping.Protocol, mapping.LocalPort, mapping.RemotePort)

	// Discover our network information
	networkInfo, err := discoverNetworkInfo(config.STUNServer)
	if err != nil {
		log.Fatalf("Failed to discover network info: %v", err)
	}

	// Create signaling client
	signalingClient := NewSignalingClient()
	defer signalingClient.Close()

	// For client, we use server's room key format
	roomKey := config.RoomID + "-server"
	networkData := formatNetworkInfo(networkInfo)
	
	// Post our network info to signaling server
	err = signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, networkData)
	if err != nil {
		log.Fatalf("Failed to post signal: %v", err)
	}

	// Wait for server network info
	serverNetworkData, err := signalingClient.WaitForPeerData(ctx, config.SignalingURL, 
		peerRole(config.Mode), roomKey, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to get server network info: %v", err)
	}

	// Parse server network info
	serverInfo := parseNetworkInfo(serverNetworkData)
	
	// Determine best connection method (LAN vs WAN)
	var targetAddr string
	if networkInfo.PrivateAddr != "" && serverInfo.PrivateAddr != "" &&
		isLANAddress(networkInfo.PrivateAddr, serverInfo.PrivateAddr) {
		// Use LAN connection
		targetAddr = serverInfo.PrivateAddr + ":" + strconv.Itoa(mapping.RemotePort)
		log.Printf("Using LAN connection to %s", targetAddr)
	} else {
		// Use WAN connection
		host, _, err := net.SplitHostPort(serverInfo.PublicAddr)
		if err != nil {
			log.Fatalf("Invalid server public address format %s: %v", serverInfo.PublicAddr, err)
		}
		targetAddr = host + ":" + strconv.Itoa(mapping.RemotePort)
		log.Printf("Using WAN connection to %s", targetAddr)
	}

	// Parse target address
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		log.Fatalf("Invalid target address format %s: %v", targetAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid target port %s: %v", portStr, err)
	}

	// Client always listens locally and connects to server
	if mapping.Protocol == "tcp" {
		runTCPClient(ctx, mapping.LocalPort, host, port)
	} else {
		runUDPClient(ctx, mapping.LocalPort, host, port)
	}
}

// parseNetworkInfo parses network info from signaling data
func parseNetworkInfo(data string) *NetworkInfo {
	info := &NetworkInfo{}
	parts := strings.Split(data, "|")
	if len(parts) >= 1 {
		info.PublicAddr = parts[0]
	}
	if len(parts) >= 2 {
		info.PrivateAddr = parts[1]
	}
	return info
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

// handleServerMode handles server mode (accepts any incoming connections)
func handleServerMode(ctx context.Context, config Configuration) {
	log.Printf("[%s] Starting server mode, ready to accept connections", config.Mode)

	// Discover network information
	networkInfo, err := discoverNetworkInfo(config.STUNServer)
	if err != nil {
		log.Fatalf("Failed to discover network info: %v", err)
	}

	// Create signaling client
	signalingClient := NewSignalingClient()
	defer signalingClient.Close()

	// Post our network info to signaling server
	roomKey := config.RoomID + "-server"
	networkData := formatNetworkInfo(networkInfo)
	err = signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, networkData)
	if err != nil {
		log.Fatalf("Failed to post signal: %v", err)
	}

	log.Printf("Server waiting for client connections...")
	// Server just waits - specific port handling happens when client connects
	select {}
}

// discoverNetworkInfo discovers both public and private network information
func discoverNetworkInfo(stunServer string) (*NetworkInfo, error) {
	info := &NetworkInfo{}

	// Get private IP
	privateIP, err := getPrivateIP()
	if err != nil {
		log.Printf("Warning: Could not get private IP: %v", err)
	} else {
		info.PrivateAddr = privateIP
	}

	// Get public IP via STUN
	publicAddr, err := getPublicIP(stunServer, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	info.PublicAddr = publicAddr

	log.Printf("Network info - Private: %s, Public: %s", info.PrivateAddr, info.PublicAddr)
	return info, nil
}

// getPrivateIP gets the local private IP address
func getPrivateIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// isLANAddress checks if two addresses are in the same LAN
func isLANAddress(addr1, addr2 string) bool {
	ip1 := net.ParseIP(extractIP(addr1))
	ip2 := net.ParseIP(extractIP(addr2))
	
	if ip1 == nil || ip2 == nil {
		return false
	}

	// Check if both are private IPs and in same /24 subnet
	if isPrivateIP(ip1) && isPrivateIP(ip2) {
		return ip1.Mask(net.CIDRMask(24, 32)).Equal(ip2.Mask(net.CIDRMask(24, 32)))
	}
	
	return false
}

// extractIP extracts IP from "ip:port" format
func extractIP(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// isPrivateIP checks if IP is in private ranges
func isPrivateIP(ip net.IP) bool {
	private := []string{
		"10.0.0.0/8",
		"172.16.0.0/12", 
		"192.168.0.0/16",
	}
	
	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// formatNetworkInfo formats network info for signaling
func formatNetworkInfo(info *NetworkInfo) string {
	return info.PublicAddr + "|" + info.PrivateAddr
}