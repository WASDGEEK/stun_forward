// Package main - STUN discovery with caching support
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
)

// stunCache caches STUN discovery results
type stunCache struct {
	publicAddr string
	timestamp  time.Time
	mutex      sync.RWMutex
}

var globalSTUNCache = &stunCache{}

// NATType represents different types of NAT
type NATType int

const (
	NATTypeUnknown NATType = iota
	NATTypeNone             // No NAT (direct internet connection)
	NATTypeFullCone         // Full Cone NAT (easiest to traverse)
	NATTypeRestrictedCone   // Restricted Cone NAT  
	NATTypePortRestricted   // Port Restricted Cone NAT
	NATTypeSymmetric       // Symmetric NAT (hardest to traverse)
)

func (nt NATType) String() string {
	switch nt {
	case NATTypeNone:
		return "No NAT"
	case NATTypeFullCone:
		return "Full Cone NAT"
	case NATTypeRestrictedCone:
		return "Restricted Cone NAT"
	case NATTypePortRestricted:
		return "Port Restricted Cone NAT"
	case NATTypeSymmetric:
		return "Symmetric NAT"
	default:
		return "Unknown NAT"
	}
}

// STUNResult contains comprehensive STUN discovery results
type STUNResult struct {
	PublicAddr  string
	LocalAddr   string
	NATType     NATType
	Mappings    []string // Different external mappings for symmetric NAT detection
	CanHolePunch bool    // Whether hole punching is likely to work
}

// getPublicIP discovers public IP address with caching support, trying both IPv4 and IPv6
func getPublicIP(stunServer string, cacheDuration time.Duration) (string, error) {
	// 先检查缓存
	globalSTUNCache.mutex.RLock()
	if time.Since(globalSTUNCache.timestamp) < cacheDuration && globalSTUNCache.publicAddr != "" {
		addr := globalSTUNCache.publicAddr
		globalSTUNCache.mutex.RUnlock()
		return addr, nil
	}
	globalSTUNCache.mutex.RUnlock()

	// 缓存过期或不存在，重新获取 - 同时尝试IPv4和IPv6
	publicAddr, err := performDualStackSTUNDiscovery(stunServer)
	if err != nil {
		return "", err
	}

	// 更新缓存
	globalSTUNCache.mutex.Lock()
	globalSTUNCache.publicAddr = publicAddr
	globalSTUNCache.timestamp = time.Now()
	globalSTUNCache.mutex.Unlock()

	return publicAddr, nil
}

// performDualStackSTUNDiscovery tries both IPv4 and IPv6 STUN discovery
func performDualStackSTUNDiscovery(stunServer string) (string, error) {
	// Try IPv4 first (usually more reliable)
	if addr, err := performSTUNDiscoveryWithNetwork(stunServer, "udp4"); err == nil {
		return addr, nil
	}
	
	// If IPv4 fails, try IPv6
	if addr, err := performSTUNDiscoveryWithNetwork(stunServer, "udp6"); err == nil {
		return addr, nil
	}
	
	// If both fail, try original method (let system decide)
	return performSTUNDiscovery(stunServer)
}

// performSTUNDiscoveryWithNetwork performs STUN discovery with specific network type
func performSTUNDiscoveryWithNetwork(stunServer, network string) (string, error) {
	// Create a new UDP connection to the STUN server with specific network type
	conn, err := net.Dial(network, stunServer)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create a new STUN client
	client, err := stun.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a binding request
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	var publicAddr string
	// The callback function will be called when a response is received
	callback := func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			return
		}
		publicAddr = xorAddr.String()
	}

	// Send the request and wait for the response
	if err = client.Do(message, callback); err != nil {
		return "", err
	}

	if publicAddr == "" {
		return "", errors.New("failed to get public IP from STUN server")
	}

	return publicAddr, nil
}

// performSTUNDiscovery performs actual STUN discovery
func performSTUNDiscovery(stunServer string) (string, error) {
	// Create a new UDP connection to the STUN server.
	conn, err := net.Dial("udp", stunServer)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create a new STUN client.
	client, err := stun.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a binding request.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	var publicAddr string
	// The callback function will be called when a response is received.
	callback := func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			return
		}
		publicAddr = xorAddr.String()
	}

	// Send the request and wait for the response.
	if err = client.Do(message, callback); err != nil {
		return "", err
	}

	if publicAddr == "" {
		return "", errors.New("failed to get public IP from STUN server")
	}

	return publicAddr, nil
}

// clearSTUNCache clears STUN cache for testing or forced refresh
func clearSTUNCache() {
	globalSTUNCache.mutex.Lock()
	globalSTUNCache.publicAddr = ""
	globalSTUNCache.timestamp = time.Time{}
	globalSTUNCache.mutex.Unlock()
}

// discoverNATType performs comprehensive NAT type detection
func discoverNATType(primarySTUN, secondarySTUN string) (*STUNResult, error) {
	result := &STUNResult{
		NATType: NATTypeUnknown,
		Mappings: make([]string, 0),
	}

	// Step 1: Get local address
	localConn, err := net.Dial("udp", primarySTUN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary STUN server: %w", err)
	}
	result.LocalAddr = localConn.LocalAddr().String()
	localConn.Close()

	log.Printf("NAT Detection - Local address: %s", result.LocalAddr)

	// Step 2: Test 1 - Basic STUN discovery
	mapping1, err := performSTUNDiscovery(primarySTUN)
	if err != nil {
		return nil, fmt.Errorf("primary STUN discovery failed: %w", err)
	}
	result.PublicAddr = mapping1
	result.Mappings = append(result.Mappings, mapping1)

	log.Printf("NAT Detection - Primary mapping: %s", mapping1)

	// Check if we have no NAT (local == public IP)
	localIP := extractIP(result.LocalAddr)
	publicIP := extractIP(mapping1)
	if localIP == publicIP {
		result.NATType = NATTypeNone
		result.CanHolePunch = true
		log.Printf("NAT Detection - No NAT detected (direct connection)")
		return result, nil
	}

	// Step 3: Test 2 - Same server, different port (symmetric NAT detection)
	mapping2, err := performSTUNDiscoveryFromSameLocalPort(primarySTUN, result.LocalAddr)
	if err != nil {
		log.Printf("Secondary mapping test failed: %v", err)
		// Continue with limited detection
	} else {
		result.Mappings = append(result.Mappings, mapping2)
		log.Printf("NAT Detection - Secondary mapping: %s", mapping2)

		// If mappings are different, it's symmetric NAT
		if mapping1 != mapping2 {
			result.NATType = NATTypeSymmetric
			result.CanHolePunch = false
			log.Printf("NAT Detection - Symmetric NAT detected (different mappings)")
			return result, nil
		}
	}

	// Step 4: Test 3 - Different server (cone NAT type detection)
	if secondarySTUN != "" && secondarySTUN != primarySTUN {
		mapping3, err := performSTUNDiscovery(secondarySTUN)
		if err != nil {
			log.Printf("Secondary STUN server test failed: %v", err)
		} else {
			result.Mappings = append(result.Mappings, mapping3)
			log.Printf("NAT Detection - Different server mapping: %s", mapping3)

			// Same mapping across servers suggests Full Cone NAT
			if extractPort(mapping1) == extractPort(mapping3) {
				result.NATType = NATTypeFullCone
				result.CanHolePunch = true
				log.Printf("NAT Detection - Full Cone NAT detected")
				return result, nil
			}
		}
	}

	// Default to Restricted Cone NAT (most common)
	if result.NATType == NATTypeUnknown {
		result.NATType = NATTypeRestrictedCone
		result.CanHolePunch = true
		log.Printf("NAT Detection - Assuming Restricted Cone NAT")
	}

	return result, nil
}

// performSTUNDiscoveryFromSameLocalPort performs STUN discovery using specific local port
func performSTUNDiscoveryFromSameLocalPort(stunServer, localAddr string) (string, error) {
	// Parse local address to get IP and port
	localIP, localPortStr, err := net.SplitHostPort(localAddr)
	if err != nil {
		return "", fmt.Errorf("invalid local address: %w", err)
	}

	// Create connection with same local address
	localUDPAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(localIP, localPortStr))
	if err != nil {
		return "", fmt.Errorf("failed to resolve local UDP address: %w", err)
	}

	conn, err := net.DialUDP("udp", localUDPAddr, nil)
	if err != nil {
		// Try with system-assigned port if exact port fails
		genericConn, err2 := net.Dial("udp", stunServer)
		if err2 != nil {
			return "", fmt.Errorf("failed to create UDP connection: %w", err2)
		}
		conn = genericConn.(*net.UDPConn)
	}
	defer conn.Close()

	// Perform STUN discovery using this connection
	client, err := stun.NewClient(conn)
	if err != nil {
		return "", fmt.Errorf("failed to create STUN client: %w", err)
	}
	defer client.Close()

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var publicAddr string

	callback := func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			return
		}
		publicAddr = xorAddr.String()
	}

	if err = client.Do(message, callback); err != nil {
		return "", fmt.Errorf("STUN request failed: %w", err)
	}

	if publicAddr == "" {
		return "", errors.New("no public address received from STUN server")
	}

	return publicAddr, nil
}

// extractPort extracts port from "ip:port" format
func extractPort(addr string) string {
	if _, port, err := net.SplitHostPort(addr); err == nil {
		return port
	}
	return ""
}

// createHolePunchingConn creates a UDP connection optimized for hole punching
func createHolePunchingConn(localAddr string) (*net.UDPConn, error) {
	if localAddr == "" {
		// Use system-assigned port
		addr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			return nil, err
		}
		return net.ListenUDP("udp", addr)
	}

	// Use specific local address for consistent hole punching
	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", addr)
}