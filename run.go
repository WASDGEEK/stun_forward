// Package main - Main runner for P2P port forwarding
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	if config.Mode == "client" {
		// Client mode: handle each mapping
		for _, mapping := range config.Mappings {
			go handlePortMapping(ctx, config, mapping)
		}
	} else {
		// Server mode: continuous polling for connections
		go handleServerMode(ctx, config)
	}
	
	// Wait for shutdown signal
	<-sigChan
	log.Println("\\nReceived shutdown signal, stopping...")
	cancel()
	
	// Give goroutines a moment to clean up
	time.Sleep(500 * time.Millisecond)
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
	isLAN := detectLANConnection(networkInfo, serverInfo)
	
	if isLAN {
		// Use LAN connection
		targetAddr = extractIP(serverInfo.PrivateAddr) + ":" + strconv.Itoa(mapping.RemotePort)
		log.Printf("🏠 Using LAN connection to %s (same network detected)", targetAddr)
	} else {
		// Use WAN connection
		host, _, err := net.SplitHostPort(serverInfo.PublicAddr)
		if err != nil {
			log.Fatalf("Invalid server public address format %s: %v", serverInfo.PublicAddr, err)
		}
		targetAddr = host + ":" + strconv.Itoa(mapping.RemotePort)
		log.Printf("🌐 Using WAN connection to %s (different networks)", targetAddr)
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

// handleServerMode handles server mode with continuous polling for client connections
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
	log.Printf("Press Ctrl+C to stop the server")

	// Continuous polling for client connection requests
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Server shutting down...")
			return
		case <-ticker.C:
			// Refresh our presence in the signaling server
			err := signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, networkData)
			if err != nil {
				log.Printf("Warning: Failed to refresh server presence: %v", err)
			} else {
				log.Printf("Server presence refreshed")
			}
		}
	}
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

// isLANAddress checks if two addresses are in the same LAN using multiple strategies
func isLANAddress(addr1, addr2 string) bool {
	ip1 := net.ParseIP(extractIP(addr1))
	ip2 := net.ParseIP(extractIP(addr2))
	
	if ip1 == nil || ip2 == nil {
		return false
	}

	// Only check if both are private IPs
	if !isPrivateIP(ip1) || !isPrivateIP(ip2) {
		return false
	}

	// Strategy 1: Same /24 subnet (most precise)
	if ip1.Mask(net.CIDRMask(24, 32)).Equal(ip2.Mask(net.CIDRMask(24, 32))) {
		return true
	}

	// Strategy 2: Same /16 subnet (192.168.x.x range)
	if isIn192168Range(ip1) && isIn192168Range(ip2) {
		if ip1.Mask(net.CIDRMask(16, 32)).Equal(ip2.Mask(net.CIDRMask(16, 32))) {
			return true
		}
	}

	// Strategy 3: Same /8 subnet (10.x.x.x range)  
	if isIn10Range(ip1) && isIn10Range(ip2) {
		if ip1.Mask(net.CIDRMask(8, 32)).Equal(ip2.Mask(net.CIDRMask(8, 32))) {
			return true
		}
	}

	// Strategy 4: Same /12 subnet (172.16-31.x.x range)
	if isIn172Range(ip1) && isIn172Range(ip2) {
		if ip1.Mask(net.CIDRMask(12, 32)).Equal(ip2.Mask(net.CIDRMask(12, 32))) {
			return true
		}
	}
	
	return false
}

// isIn192168Range checks if IP is in 192.168.0.0/16 range
func isIn192168Range(ip net.IP) bool {
	_, network, _ := net.ParseCIDR("192.168.0.0/16")
	return network.Contains(ip)
}

// isIn10Range checks if IP is in 10.0.0.0/8 range
func isIn10Range(ip net.IP) bool {
	_, network, _ := net.ParseCIDR("10.0.0.0/8")
	return network.Contains(ip)
}

// isIn172Range checks if IP is in 172.16.0.0/12 range
func isIn172Range(ip net.IP) bool {
	_, network, _ := net.ParseCIDR("172.16.0.0/12")
	return network.Contains(ip)
}

// detectLANConnection uses multiple strategies to detect if two devices are on the same LAN
func detectLANConnection(clientInfo, serverInfo *NetworkInfo) bool {
	// Strategy 1: Public IP comparison (most reliable for NAT detection)
	if clientInfo.PublicAddr != "" && serverInfo.PublicAddr != "" {
		clientPublicIP := extractIP(clientInfo.PublicAddr)
		serverPublicIP := extractIP(serverInfo.PublicAddr)
		
		if clientPublicIP == serverPublicIP {
			log.Printf("🔍 LAN detected: Same public IP (%s)", clientPublicIP)
			return true
		}
	}
	
	// Strategy 2: Private IP subnet analysis
	if clientInfo.PrivateAddr != "" && serverInfo.PrivateAddr != "" {
		if isLANAddress(clientInfo.PrivateAddr, serverInfo.PrivateAddr) {
			log.Printf("🔍 LAN detected: Same private subnet (%s <-> %s)", 
				extractIP(clientInfo.PrivateAddr), extractIP(serverInfo.PrivateAddr))
			return true
		}
	}
	
	log.Printf("🔍 WAN detected: Different networks (Public: %s vs %s, Private: %s vs %s)",
		extractIP(clientInfo.PublicAddr), extractIP(serverInfo.PublicAddr),
		extractIP(clientInfo.PrivateAddr), extractIP(serverInfo.PrivateAddr))
	
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
	// Add a default port to private IP if it doesn't have one
	privateAddr := info.PrivateAddr
	if privateAddr != "" && !strings.Contains(privateAddr, ":") {
		privateAddr = privateAddr + ":0" // Add port 0 as placeholder
	}
	return info.PublicAddr + "|" + privateAddr
}