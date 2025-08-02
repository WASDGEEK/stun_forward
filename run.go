// Package main - Main runner for P2P port forwarding
package main

import (
	"context"
	"encoding/json"
	"fmt"
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
		// Client mode: register once and handle all mappings
		go handleClientMode(ctx, config)
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

// handleClientMode handles client mode - register once and handle all mappings
func handleClientMode(ctx context.Context, config Configuration) {
	log.Printf("[%s] Starting client mode with %d mappings", config.Mode, len(config.Mappings))

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
	
	// Format client registration data including mappings
	clientData, err := formatClientRegistrationData(networkInfo, config.Mappings)
	if err != nil {
		log.Fatalf("Failed to format client registration data: %v", err)
	}
	
	// Debug: Print what client is sending
	log.Printf("DEBUG: Client mode: %s", config.Mode)
	log.Printf("DEBUG: Room key: %s", roomKey)
	log.Printf("DEBUG: Sending client registration data: %q", clientData)
	log.Printf("DEBUG: Data length: %d", len(clientData))
	
	// Post our network info and mappings to signaling server
	err = signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, clientData)
	if err != nil {
		log.Fatalf("Failed to post signal: %v", err)
	}

	// Wait for server registration data with retry mechanism
	var serverData *ServerRegistrationData
	maxRetries := 5
	retryDelay := 2 * time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Waiting for server port allocation data (attempt %d/%d)...", attempt, maxRetries)
		
		serverRegistrationData, err := signalingClient.WaitForPeerData(ctx, config.SignalingURL, 
			peerRole(config.Mode), roomKey, 15*time.Second)
		if err != nil {
			log.Printf("Attempt %d failed to get server data: %v", attempt, err)
			if attempt == maxRetries {
				log.Fatalf("Failed to get server registration data after %d attempts", maxRetries)
			}
			time.Sleep(retryDelay)
			continue
		}

		// Debug: Print raw server registration data
		log.Printf("DEBUG: Received raw server data (attempt %d): %q", attempt, serverRegistrationData)
		log.Printf("DEBUG: Server data length: %d", len(serverRegistrationData))
		
		// Check if it's old format (server hasn't finished port allocation yet)
		if strings.Contains(serverRegistrationData, "|") && !strings.HasPrefix(serverRegistrationData, "{") {
			log.Printf("Server still sending initial data, port allocation not ready yet (attempt %d)", attempt)
			if attempt == maxRetries {
				log.Fatalf("Server never sent port allocation data after %d attempts", maxRetries)
			}
			time.Sleep(retryDelay)
			continue
		}
		
		// Try to parse server registration data
		serverData, err = parseServerRegistrationData(serverRegistrationData)
		if err != nil {
			log.Printf("Failed to parse server data (attempt %d): %v", attempt, err)
			log.Printf("Raw server data was: %q", serverRegistrationData)
			if attempt == maxRetries {
				log.Fatalf("Failed to parse server registration data after %d attempts", maxRetries)
			}
			time.Sleep(retryDelay)
			continue
		}
		
		// Success!
		log.Printf("Successfully received server port allocation data on attempt %d", attempt)
		break
	}

	log.Printf("Received server port allocations for %d mappings", len(serverData.PortMappings))
	
	// Start port forwarding for each mapping with allocated ports
	for _, portMapping := range serverData.PortMappings {
		clientMapping := portMapping.ClientMapping
		allocatedPort := portMapping.AllocatedPort
		
		log.Printf("Server allocated port %d for client mapping %d->%d", 
			allocatedPort, clientMapping.LocalPort, clientMapping.RemotePort)
		
		go handlePortMappingWithAllocatedPort(ctx, config, clientMapping, allocatedPort, 
			networkInfo, &serverData.NetworkInfo)
	}
	
	// Keep client alive
	<-ctx.Done()
	log.Printf("Client shutting down...")
}

// handlePortMappingWithAllocatedPort handles a single port mapping with server's allocated port
func handlePortMappingWithAllocatedPort(ctx context.Context, config Configuration, mapping PortMapping, 
	allocatedPort int, clientInfo, serverInfo *NetworkInfo) {
	log.Printf("[%s] Starting port forward: %s %d -> allocated port %d", 
		config.Mode, mapping.Protocol, mapping.LocalPort, allocatedPort)
	
	// Determine best connection method (LAN vs WAN)
	var targetAddr string
	isLAN := detectLANConnection(clientInfo, serverInfo)
	
	if isLAN {
		// Use LAN connection to allocated port
		targetAddr = extractIP(serverInfo.PrivateAddr) + ":" + strconv.Itoa(allocatedPort)
		log.Printf("🏠 Using LAN connection to %s (same network detected)", targetAddr)
	} else {
		// Use WAN connection to allocated port
		host, _, err := net.SplitHostPort(serverInfo.PublicAddr)
		if err != nil {
			log.Fatalf("Invalid server public address format %s: %v", serverInfo.PublicAddr, err)
		}
		targetAddr = host + ":" + strconv.Itoa(allocatedPort)
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

	// Client always listens locally and connects to server's allocated port
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

// allocatePortForMapping dynamically allocates a port for the mapping
func allocatePortForMapping(ctx context.Context, mapping PortMapping) (int, error) {
	var ln net.Listener
	var err error
	
	if mapping.Protocol == "tcp" {
		ln, err = net.Listen("tcp", ":0")
	} else {
		// For UDP, we need to use a different approach
		addr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			return 0, err
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return 0, err
		}
		port := conn.LocalAddr().(*net.UDPAddr).Port
		conn.Close()
		return port, nil
	}
	
	if err != nil {
		return 0, fmt.Errorf("failed to allocate port for %s: %w", mapping.Protocol, err)
	}
	
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port, nil
}

// handleServerMode handles server mode - dynamic port allocation and forwarding
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

	// Don't post initial data - wait for client first to avoid overwriting
	roomKey := config.RoomID + "-server"
	
	// Debug: Print server setup
	log.Printf("DEBUG: Server mode: %s", config.Mode)
	log.Printf("DEBUG: Room key: %s", roomKey)
	
	log.Printf("Server waiting for client connections...")
	log.Printf("Waiting for client to register with mapping configuration...")

	// Wait for client registration data (including mappings)
	clientRegistrationData, err := signalingClient.WaitForPeerData(ctx, config.SignalingURL, 
		"client", roomKey, 60*time.Second)
	if err != nil {
		log.Fatalf("Failed to get client registration data: %v", err)
	}

	// Debug: Print raw client registration data
	log.Printf("DEBUG: Received raw client data: %q", clientRegistrationData)
	log.Printf("DEBUG: Client data length: %d", len(clientRegistrationData))
	
	// Parse client registration data
	clientData, err := parseClientRegistrationData(clientRegistrationData)
	if err != nil {
		log.Printf("ERROR: Failed to parse client registration data: %v", err)
		log.Printf("ERROR: Raw data was: %q", clientRegistrationData)
		
		// Try to detect if it's old format (network info string)
		if strings.Contains(clientRegistrationData, "|") && !strings.HasPrefix(clientRegistrationData, "{") {
			log.Printf("ERROR: Detected old network info format. Client might be using old version.")
		}
		log.Fatalf("Client registration parsing failed")
	}

	log.Printf("Received client registration with %d mappings", len(clientData.Mappings))
	
	// Parse mapping strings back to PortMapping structs
	var parsedMappings []PortMapping
	for _, mappingStr := range clientData.Mappings {
		var mapping PortMapping
		err := mapping.parseFromString(mappingStr)
		if err != nil {
			log.Fatalf("Failed to parse mapping string %q: %v", mappingStr, err)
		}
		parsedMappings = append(parsedMappings, mapping)
	}
	
	// Allocate dynamic ports for each mapping
	var portMappings []ServerPortMapping
	for _, mapping := range parsedMappings {
		allocatedPort, err := allocatePortForMapping(ctx, mapping)
		if err != nil {
			log.Fatalf("Failed to allocate port for mapping %+v: %v", mapping, err)
		}
		
		portMapping := ServerPortMapping{
			ClientMapping: mapping,
			AllocatedPort: allocatedPort,
		}
		portMappings = append(portMappings, portMapping)
		
		log.Printf("Allocated %s port %d for client mapping %d->%d", 
			mapping.Protocol, allocatedPort, mapping.LocalPort, mapping.RemotePort)
	}

	// Send port allocation results back to client
	serverData, err := formatServerRegistrationData(networkInfo, portMappings)
	if err != nil {
		log.Fatalf("Failed to format server registration data: %v", err)
	}
	
	// Debug: Print what server is sending as final registration
	log.Printf("DEBUG: Sending final server registration data: %q", serverData)
	log.Printf("DEBUG: Final data length: %d", len(serverData))
	
	err = signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, serverData)
	if err != nil {
		log.Fatalf("Failed to post server registration data: %v", err)
	}
	
	log.Printf("Server port allocation data sent to signaling server")

	// Start port listeners for each allocated port
	for _, portMapping := range portMappings {
		mapping := portMapping.ClientMapping
		allocatedPort := portMapping.AllocatedPort
		
		log.Printf("Starting %s server on allocated port %d -> local service 127.0.0.1:%d", 
			mapping.Protocol, allocatedPort, mapping.RemotePort)
		
		if mapping.Protocol == "tcp" {
			go runTCPServerOnPort(ctx, allocatedPort, mapping.RemotePort)
		} else {
			go runUDPServerOnPort(ctx, allocatedPort, mapping.RemotePort)
		}
	}

	log.Printf("Server ready! All %d port listeners started.", len(portMappings))
	log.Printf("Press Ctrl+C to stop the server")

	// Keep server alive and periodically refresh presence
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Server shutting down...")
			return
		case <-ticker.C:
			// Refresh server registration data
			err := signalingClient.PostSignal(config.SignalingURL, config.Mode, roomKey, serverData)
			if err != nil {
				log.Printf("Warning: Failed to refresh server presence: %v", err)
			} else {
				log.Printf("Server presence refreshed with %d port mappings", len(portMappings))
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

// formatNetworkInfo formats network info for signaling (server only)
func formatNetworkInfo(info *NetworkInfo) string {
	// Add a default port to private IP if it doesn't have one
	privateAddr := info.PrivateAddr
	if privateAddr != "" && !strings.Contains(privateAddr, ":") {
		privateAddr = privateAddr + ":0" // Add port 0 as placeholder
	}
	return info.PublicAddr + "|" + privateAddr
}

// formatClientRegistrationData formats client registration data including mappings
func formatClientRegistrationData(info *NetworkInfo, mappings []PortMapping) (string, error) {
	// Convert PortMapping structs to string format
	var mappingStrings []string
	for _, mapping := range mappings {
		mappingStr := fmt.Sprintf("%s:%d:%d", mapping.Protocol, mapping.LocalPort, mapping.RemotePort)
		mappingStrings = append(mappingStrings, mappingStr)
	}
	
	clientData := ClientRegistrationData{
		NetworkInfo: *info,
		Mappings:    mappingStrings,
	}
	
	jsonData, err := json.Marshal(clientData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal client registration data: %w", err)
	}
	return string(jsonData), nil
}

// parseClientRegistrationData parses client registration data from JSON
func parseClientRegistrationData(data string) (*ClientRegistrationData, error) {
	var clientData ClientRegistrationData
	err := json.Unmarshal([]byte(data), &clientData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal client registration data: %w", err)
	}
	return &clientData, nil
}

// formatServerRegistrationData formats server registration data including port mappings
func formatServerRegistrationData(info *NetworkInfo, portMappings []ServerPortMapping) (string, error) {
	serverData := ServerRegistrationData{
		NetworkInfo:  *info,
		PortMappings: portMappings,
	}
	
	jsonData, err := json.Marshal(serverData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal server registration data: %w", err)
	}
	return string(jsonData), nil
}

// parseServerRegistrationData parses server registration data from JSON
func parseServerRegistrationData(data string) (*ServerRegistrationData, error) {
	var serverData ServerRegistrationData
	err := json.Unmarshal([]byte(data), &serverData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal server registration data: %w", err)
	}
	return &serverData, nil
}