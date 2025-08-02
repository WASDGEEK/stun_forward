// Package main - UDP hole punching implementation
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// HolePunchResult represents the result of a hole punching attempt
type HolePunchResult struct {
	Success    bool
	LocalAddr  string
	RemoteAddr string
	Conn       *net.UDPConn
	Error      error
}

// HolePunchConfig contains configuration for hole punching
type HolePunchConfig struct {
	LocalSTUNAddr  string        // Our STUN-discovered address
	RemoteSTUNAddr string        // Peer's STUN-discovered address
	LocalPrivateAddr string      // Our private address
	RemotePrivateAddr string     // Peer's private address
	Timeout        time.Duration // Hole punching timeout
	RetryCount     int           // Number of retry attempts
	IsInitiator    bool          // Whether we initiate the connection
}

// performUDPHolePunching attempts UDP hole punching using multiple strategies
func performUDPHolePunching(ctx context.Context, config HolePunchConfig) (*HolePunchResult, error) {
	log.Printf("🚀 Starting UDP hole punching - Initiator: %v", config.IsInitiator)
	log.Printf("   Local STUN: %s, Remote STUN: %s", config.LocalSTUNAddr, config.RemoteSTUNAddr)
	log.Printf("   Local Private: %s, Remote Private: %s", config.LocalPrivateAddr, config.RemotePrivateAddr)

	// Strategy 1: Try direct connection to STUN addresses (most common)
	if result := tryDirectConnection(ctx, config.LocalSTUNAddr, config.RemoteSTUNAddr, config.Timeout); result.Success {
		log.Printf("✅ Hole punching successful via STUN addresses")
		return result, nil
	}

	// Strategy 2: Simultaneous UDP hole punching
	if result := trySimultaneousConnect(ctx, config); result.Success {
		log.Printf("✅ Hole punching successful via simultaneous connect")
		return result, nil
	}

	// Strategy 3: Sequential port prediction (for symmetric NAT)
	if result := tryPortPrediction(ctx, config); result.Success {
		log.Printf("✅ Hole punching successful via port prediction")
		return result, nil
	}

	// Strategy 4: Try private addresses (LAN fallback)
	if config.LocalPrivateAddr != "" && config.RemotePrivateAddr != "" {
		if result := tryDirectConnection(ctx, config.LocalPrivateAddr, config.RemotePrivateAddr, config.Timeout); result.Success {
			log.Printf("✅ Direct LAN connection successful")
			return result, nil
		}
	}

	return &HolePunchResult{
		Success: false,
		Error:   fmt.Errorf("all hole punching strategies failed"),
	}, nil
}

// tryDirectConnection attempts a direct UDP connection
func tryDirectConnection(ctx context.Context, localAddr, remoteAddr string, timeout time.Duration) *HolePunchResult {
	log.Printf("🎯 Trying direct connection: %s -> %s", localAddr, remoteAddr)

	// Parse addresses
	localUDPAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("invalid local address: %w", err)}
	}

	remoteUDPAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("invalid remote address: %w", err)}
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", localUDPAddr)
	if err != nil {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("failed to listen UDP: %w", err)}
	}

	// Set timeout
	deadline := time.Now().Add(timeout)
	conn.SetDeadline(deadline)

	// Send initial packet to open NAT mapping
	testMessage := []byte("HOLE_PUNCH_INIT")
	_, err = conn.WriteToUDP(testMessage, remoteUDPAddr)
	if err != nil {
		conn.Close()
		return &HolePunchResult{Success: false, Error: fmt.Errorf("failed to send init packet: %w", err)}
	}

	// Try to receive response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, addr, err := conn.ReadFromUDP(buffer)
	if err == nil && n > 0 {
		log.Printf("   Received response from %s: %s", addr, string(buffer[:n]))
		conn.SetDeadline(time.Time{}) // Clear deadline
		return &HolePunchResult{
			Success:    true,
			LocalAddr:  conn.LocalAddr().String(),
			RemoteAddr: addr.String(),
			Conn:       conn,
		}
	}

	conn.Close()
	return &HolePunchResult{Success: false, Error: fmt.Errorf("no response received")}
}

// trySimultaneousConnect attempts simultaneous UDP connection from both sides
func trySimultaneousConnect(ctx context.Context, config HolePunchConfig) *HolePunchResult {
	log.Printf("🔄 Trying simultaneous connect")

	// Parse remote address
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", config.RemoteSTUNAddr)
	if err != nil {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("invalid remote address: %w", err)}
	}

	// Create connection using same local port as STUN discovery
	localIP := extractIP(config.LocalSTUNAddr)
	localPort := extractPort(config.LocalSTUNAddr)
	localAddr := net.JoinHostPort(localIP, localPort)
	
	localUDPAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("invalid local address: %w", err)}
	}

	conn, err := net.ListenUDP("udp", localUDPAddr)
	if err != nil {
		// Try with system-assigned port if specific port fails
		conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: localUDPAddr.IP})
		if err != nil {
			return &HolePunchResult{Success: false, Error: fmt.Errorf("failed to listen UDP: %w", err)}
		}
	}

	// Simultaneous connect pattern
	var wg sync.WaitGroup
	var result *HolePunchResult
	var mutex sync.Mutex

	// Goroutine 1: Keep sending packets
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		timeout := time.After(config.Timeout)
		message := []byte("SIMULTANEOUS_CONNECT")
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeout:
				return
			case <-ticker.C:
				conn.WriteToUDP(message, remoteUDPAddr)
			}
		}
	}()

	// Goroutine 2: Listen for responses
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		buffer := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(config.Timeout))
		
		for {
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			
			if n > 0 {
				log.Printf("   Simultaneous connect response from %s: %s", addr, string(buffer[:n]))
				
				mutex.Lock()
				if result == nil {
					result = &HolePunchResult{
						Success:    true,
						LocalAddr:  conn.LocalAddr().String(),
						RemoteAddr: addr.String(),
						Conn:       conn,
					}
				}
				mutex.Unlock()
				return
			}
		}
	}()

	// Wait for completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(config.Timeout):
	case <-ctx.Done():
	}

	mutex.Lock()
	defer mutex.Unlock()
	
	if result != nil {
		return result
	}

	conn.Close()
	return &HolePunchResult{Success: false, Error: fmt.Errorf("simultaneous connect failed")}
}

// tryPortPrediction attempts port prediction for symmetric NAT
func tryPortPrediction(ctx context.Context, config HolePunchConfig) *HolePunchResult {
	log.Printf("🎲 Trying port prediction for symmetric NAT")

	// Extract base port from remote STUN address
	remoteIP := extractIP(config.RemoteSTUNAddr)
	basePort := extractPort(config.RemoteSTUNAddr)
	
	if basePort == "" {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("cannot extract port for prediction")}
	}

	// Convert base port to int
	basePortNum := 0
	fmt.Sscanf(basePort, "%d", &basePortNum)

	// Try a range of ports around the base port
	portRange := []int{0, 1, -1, 2, -2, 3, -3, 4, -4, 5, -5}
	
	for _, offset := range portRange {
		targetPort := basePortNum + offset
		if targetPort <= 0 || targetPort > 65535 {
			continue
		}

		targetAddr := fmt.Sprintf("%s:%d", remoteIP, targetPort)
		log.Printf("   Trying predicted port: %s", targetAddr)

		if result := tryDirectConnection(ctx, config.LocalSTUNAddr, targetAddr, 1*time.Second); result.Success {
			log.Printf("   Port prediction successful with offset %d", offset)
			return result
		}
	}

	return &HolePunchResult{Success: false, Error: fmt.Errorf("port prediction failed")}
}

// getLocalInterfaceIP gets the local interface IP address
func getLocalInterfaceIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// createReusePortUDPConn creates a UDP connection with port reuse enabled
func createReusePortUDPConn(addr *net.UDPAddr) (*net.UDPConn, error) {
	// First try regular UDP listen
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	
	// Enable socket options for better hole punching (if supported)
	if err := enableSocketReuse(conn); err != nil {
		log.Printf("Warning: failed to enable socket reuse: %v", err)
	}
	
	return conn, nil
}

// enableSocketReuse enables SO_REUSEADDR and SO_REUSEPORT if available
func enableSocketReuse(conn *net.UDPConn) error {
	// This is platform-specific, for now just return success
	// Real implementation would use syscalls to set SO_REUSEADDR/SO_REUSEPORT
	log.Printf("Socket reuse options would be enabled here (platform-specific)")
	return nil
}

// SynchronizedHolePunch performs coordinated hole punching with precise timing
type SynchronizedHolePunch struct {
	config      HolePunchConfig
	syncChannel chan struct{}
	result      *HolePunchResult
	mutex       sync.Mutex
}

// establishP2PConnection creates a P2P connection using improved hole punching
func establishP2PConnection(ctx context.Context, localInfo, remoteInfo *NetworkInfo, isInitiator bool) (*net.UDPConn, error) {
	config := HolePunchConfig{
		LocalSTUNAddr:     localInfo.PublicAddr,
		RemoteSTUNAddr:    remoteInfo.PublicAddr,
		LocalPrivateAddr:  localInfo.PrivateAddr,
		RemotePrivateAddr: remoteInfo.PrivateAddr,
		Timeout:           15 * time.Second, // Increased timeout for better success
		RetryCount:        5,                // More retries
		IsInitiator:       isInitiator,
	}

	// Improved timing coordination
	if isInitiator {
		// Initiator starts immediately but with coordination
		log.Printf("🚀 Initiator starting hole punching sequence")
	} else {
		// Non-initiator waits slightly longer for better coordination
		delay := 800 * time.Millisecond
		log.Printf("⏳ Non-initiator waiting %v for coordination", delay)
		time.Sleep(delay)
	}

	// Use synchronized hole punching for better success rate
	result, err := performSynchronizedHolePunching(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("synchronized hole punching failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("hole punching unsuccessful: %v", result.Error)
	}

	log.Printf("🎉 P2P connection established: %s <-> %s", result.LocalAddr, result.RemoteAddr)
	return result.Conn, nil
}

// performSynchronizedHolePunching performs hole punching with better timing
func performSynchronizedHolePunching(ctx context.Context, config HolePunchConfig) (*HolePunchResult, error) {
	log.Printf("🚀 Starting synchronized UDP hole punching - Initiator: %v", config.IsInitiator)
	log.Printf("   Local STUN: %s, Remote STUN: %s", config.LocalSTUNAddr, config.RemoteSTUNAddr)
	log.Printf("   Local Private: %s, Remote Private: %s", config.LocalPrivateAddr, config.RemotePrivateAddr)

	// Strategy 1: Try LAN direct connection first (fastest)
	if config.LocalPrivateAddr != "" && config.RemotePrivateAddr != "" {
		if result := tryDirectConnection(ctx, config.LocalPrivateAddr, config.RemotePrivateAddr, 2*time.Second); result.Success {
			log.Printf("✅ LAN direct connection successful")
			return result, nil
		}
	}

	// Strategy 2: Enhanced simultaneous connect with better timing
	if result := tryEnhancedSimultaneousConnect(ctx, config); result.Success {
		log.Printf("✅ Enhanced simultaneous connect successful")
		return result, nil
	}

	// Strategy 3: Try direct STUN addresses with retry
	for attempt := 0; attempt < config.RetryCount; attempt++ {
		log.Printf("🔄 Attempt %d/%d: Trying STUN addresses", attempt+1, config.RetryCount)
		if result := tryDirectConnection(ctx, config.LocalSTUNAddr, config.RemoteSTUNAddr, 3*time.Second); result.Success {
			log.Printf("✅ STUN direct connection successful on attempt %d", attempt+1)
			return result, nil
		}
		
		// Progressive delay between attempts
		if attempt < config.RetryCount-1 {
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
		}
	}

	// Strategy 4: Port prediction for symmetric NAT
	if result := tryPortPrediction(ctx, config); result.Success {
		log.Printf("✅ Port prediction successful")
		return result, nil
	}

	return &HolePunchResult{
		Success: false,
		Error:   fmt.Errorf("all synchronized hole punching strategies failed"),
	}, nil
}

// tryEnhancedSimultaneousConnect improved simultaneous connect with better coordination
func tryEnhancedSimultaneousConnect(ctx context.Context, config HolePunchConfig) *HolePunchResult {
	log.Printf("🔄 Trying enhanced simultaneous connect")

	// Parse remote address
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", config.RemoteSTUNAddr)
	if err != nil {
		return &HolePunchResult{Success: false, Error: fmt.Errorf("invalid remote address: %w", err)}
	}

	// Get local interface IP
	actualLocalIP, err := getLocalInterfaceIP()
	if err != nil {
		log.Printf("Failed to get local interface IP: %v", err)
		actualLocalIP = "0.0.0.0"
	}
	
	// Create local binding address
	localBindAddr := &net.UDPAddr{
		IP:   net.ParseIP(actualLocalIP),
		Port: 0, // Let system assign port initially
	}

	// Try to use STUN port for consistency
	if stunPort := extractPort(config.LocalSTUNAddr); stunPort != "" {
		if port, err := strconv.Atoi(stunPort); err == nil {
			localBindAddr.Port = port
		}
	}

	// Create connection
	conn, err := createReusePortUDPConn(localBindAddr)
	if err != nil {
		// Fallback to any port
		localBindAddr.Port = 0
		conn, err = createReusePortUDPConn(localBindAddr)
		if err != nil {
			return &HolePunchResult{Success: false, Error: fmt.Errorf("failed to create connection: %w", err)}
		}
	}

	// Enhanced simultaneous connect with coordinated timing
	var wg sync.WaitGroup
	var result *HolePunchResult
	var mutex sync.Mutex
	success := make(chan struct{}, 1)

	// Sender goroutine with improved timing
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Staggered start based on role
		if !config.IsInitiator {
			time.Sleep(100 * time.Millisecond) // Small offset for coordination
		}
		
		ticker := time.NewTicker(50 * time.Millisecond) // Faster sending rate
		defer ticker.Stop()
		
		timeout := time.After(config.Timeout)
		message := []byte(fmt.Sprintf("ENHANCED_HOLE_PUNCH_%v", config.IsInitiator))
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeout:
				return
			case <-success:
				return
			case <-ticker.C:
				conn.WriteToUDP(message, remoteUDPAddr)
			}
		}
	}()

	// Receiver goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		buffer := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(config.Timeout))
		
		for {
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			
			if n > 0 && addr != nil {
				log.Printf("   Enhanced simultaneous connect response from %s: %s", addr, string(buffer[:n]))
				
				mutex.Lock()
				if result == nil {
					result = &HolePunchResult{
						Success:    true,
						LocalAddr:  conn.LocalAddr().String(),
						RemoteAddr: addr.String(),
						Conn:       conn,
					}
					select {
					case success <- struct{}{}:
					default:
					}
				}
				mutex.Unlock()
				return
			}
		}
	}()

	// Wait for completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(config.Timeout):
	case <-ctx.Done():
	}

	mutex.Lock()
	defer mutex.Unlock()
	
	if result != nil {
		return result
	}

	conn.Close()
	return &HolePunchResult{Success: false, Error: fmt.Errorf("enhanced simultaneous connect failed")}
}