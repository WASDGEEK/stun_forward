// Package main - Network forwarding implementations
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	// TCPBufferSize optimized buffer size for TCP forwarding
	TCPBufferSize = 64 * 1024 // 64KB
	// UDPBufferSize optimized buffer size for UDP forwarding
	UDPBufferSize = 8 * 1024 // 8KB
)

// tcpProxy handles TCP data forwarding with optimized buffering
func tcpProxy(ctx context.Context, src, dst net.Conn, direction string) {
	defer src.Close()
	defer dst.Close()

	buf := make([]byte, TCPBufferSize)
	
	done := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(dst, src, buf)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil && err != io.EOF {
			log.Printf("TCP proxy %s error: %v", direction, err)
		}
	case <-ctx.Done():
		log.Printf("TCP proxy %s cancelled", direction)
	}
}

// runTCPClient runs TCP client forwarding (listens locally, connects to server)
func runTCPClient(ctx context.Context, localPort int, remoteIP string, remotePort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(localPort))
	if err != nil {
		log.Fatalf("TCP client listen error: %v", err)
	}
	defer ln.Close()

	log.Printf("TCP Client listening on port %d, forwarding to %s:%d", localPort, remoteIP, remotePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			log.Printf("TCP client accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			
			peer, err := net.Dial("tcp", net.JoinHostPort(remoteIP, strconv.Itoa(remotePort)))
			if err != nil {
				log.Printf("TCP client dial error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// Client to server
			go func() {
				defer wg.Done()
				tcpProxy(ctx, c, peer, "client->server")
			}()

			// Server to client
			go func() {
				defer wg.Done() 
				tcpProxy(ctx, peer, c, "server->client")
			}()

			wg.Wait()
		}(conn)
	}
}

// runTCPServer runs TCP server forwarding (accepts connections, forwards to local service)
func runTCPServer(ctx context.Context, m PortMapping, peerHost string, peerPort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.RemotePort))
	if err != nil {
		log.Fatalf("TCP server listen error: %v", err)
	}
	defer ln.Close()

	log.Printf("TCP Server listening on port %d, forwarding to local service 127.0.0.1:%d", m.RemotePort, m.LocalPort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			log.Printf("TCP server accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			local, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(m.LocalPort)))
			if err != nil {
				log.Printf("TCP server dial local service error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// Client to local service
			go func() {
				defer wg.Done()
				tcpProxy(ctx, c, local, "client->local")
			}()

			// Local service to client
			go func() {
				defer wg.Done()
				tcpProxy(ctx, local, c, "local->client")
			}()

			wg.Wait()
		}(conn)
	}
}

// runUDPClient runs UDP client forwarding (listens locally, forwards to server)
func runUDPClient(ctx context.Context, localPort int, remoteIP string, remotePort int) {
	localAddr := net.UDPAddr{Port: localPort}
	conn, err := net.ListenUDP("udp", &localAddr)
	if err != nil {
		log.Fatalf("UDP client listen error: %v", err)
	}
	defer conn.Close()

	remoteAddr := net.UDPAddr{IP: net.ParseIP(remoteIP), Port: remotePort}
	buf := make([]byte, UDPBufferSize)
	
	log.Printf("UDP Client listening on port %d, forwarding to %s:%d", localPort, remoteIP, remotePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP client read error: %v", err)
			continue
		}

		// Forward to remote server
		go func(data []byte, client *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &remoteAddr)
			if err != nil {
				log.Printf("UDP client write to remote error: %v", err)
			}
		}(buf[:n], clientAddr)
	}
}

// runUDPServer runs UDP server forwarding (accepts packets, forwards to local service)
func runUDPServer(ctx context.Context, m PortMapping, peerHost string, peerPort int) {
	localPeerAddr := net.UDPAddr{Port: m.RemotePort}
	conn, err := net.ListenUDP("udp", &localPeerAddr)
	if err != nil {
		log.Fatalf("UDP server listen error: %v", err)
	}
	defer conn.Close()

	localServiceAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: m.LocalPort}
	buf := make([]byte, UDPBufferSize)

	log.Printf("UDP Server listening on port %d, forwarding to local service 127.0.0.1:%d", m.RemotePort, m.LocalPort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, peerAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP server read error: %v", err)
			continue
		}

		// Forward to local service
		go func(data []byte, peer *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &localServiceAddr)
			if err != nil {
				log.Printf("UDP server write to local service error: %v", err)
			}
		}(buf[:n], peerAddr)
	}
}

// runTCPServerOnPort runs TCP server on specified port, forwarding to local service
func runTCPServerOnPort(ctx context.Context, listenPort, localServicePort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(listenPort))
	if err != nil {
		log.Fatalf("TCP server listen error on port %d: %v", listenPort, err)
	}
	defer ln.Close()

	log.Printf("TCP Server listening on port %d, forwarding to local service 127.0.0.1:%d", listenPort, localServicePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			log.Printf("TCP server accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			local, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(localServicePort)))
			if err != nil {
				log.Printf("TCP server dial local service error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// Client to local service
			go func() {
				defer wg.Done()
				tcpProxy(ctx, c, local, "client->local")
			}()

			// Local service to client
			go func() {
				defer wg.Done()
				tcpProxy(ctx, local, c, "local->client")
			}()

			wg.Wait()
		}(conn)
	}
}

// runUDPClientWithHolePunching runs UDP client with P2P hole punching
func runUDPClientWithHolePunching(ctx context.Context, localPort, remotePort int, clientInfo, serverInfo *NetworkInfo) error {
	log.Printf("🚀 Starting UDP hole punching client on port %d", localPort)

	// Establish P2P connection
	p2pConn, err := establishP2PConnection(ctx, clientInfo, serverInfo, true) // Client is initiator
	if err != nil {
		return fmt.Errorf("failed to establish P2P connection: %w", err)
	}
	defer p2pConn.Close()

	// Create local listener for applications
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return fmt.Errorf("failed to resolve local address: %w", err)
	}

	localConn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on local port: %w", err)
	}
	defer localConn.Close()

	log.Printf("✅ UDP hole punching established, proxying %d <-> P2P", localPort)

	// Bidirectional forwarding between local applications and P2P connection
	go udpForwardP2P(ctx, localConn, p2pConn, "local->p2p")
	go udpForwardP2P(ctx, p2pConn, localConn, "p2p->local")

	// Keep connection alive
	<-ctx.Done()
	return nil
}

// udpForwardP2P forwards UDP packets between connections
func udpForwardP2P(ctx context.Context, src, dst net.Conn, direction string) {
	buffer := make([]byte, UDPBufferSize)
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Set read timeout to avoid blocking indefinitely
		src.SetReadDeadline(time.Now().Add(1 * time.Second))
		
		n, err := src.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout is expected, continue loop
			}
			log.Printf("UDP forward %s read error: %v", direction, err)
			return
		}

		if n > 0 {
			dst.SetWriteDeadline(time.Now().Add(1 * time.Second))
			_, err = dst.Write(buffer[:n])
			if err != nil {
				log.Printf("UDP forward %s write error: %v", direction, err)
				return
			}
		}
	}
}

// runUDPServerWithHolePunching runs UDP server with P2P hole punching support
func runUDPServerWithHolePunching(ctx context.Context, listenPort, localServicePort int, clientInfo, serverInfo *NetworkInfo) error {
	log.Printf("🚀 Starting UDP hole punching server on port %d", listenPort)

	// Establish P2P connection (server is not initiator)
	p2pConn, err := establishP2PConnection(ctx, serverInfo, clientInfo, false)
	if err != nil {
		return fmt.Errorf("failed to establish P2P connection: %w", err)
	}
	defer p2pConn.Close()

	log.Printf("✅ UDP hole punching established, proxying P2P <-> local service %d", localServicePort)

	// Create connection to local service
	localServiceAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: localServicePort,
	}

	// Forward packets between P2P connection and local service
	go udpForwardToService(ctx, p2pConn, localServiceAddr, "p2p->service")

	// Keep connection alive
	<-ctx.Done()
	return nil
}

// udpForwardToService forwards UDP packets to local service
func udpForwardToService(ctx context.Context, p2pConn *net.UDPConn, serviceAddr *net.UDPAddr, direction string) {
	buffer := make([]byte, UDPBufferSize)
	
	// Create connection to local service
	serviceConn, err := net.Dial("udp", serviceAddr.String())
	if err != nil {
		log.Printf("Failed to connect to local service: %v", err)
		return
	}
	defer serviceConn.Close()

	// Start bidirectional forwarding
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Read from P2P connection
			p2pConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, err := p2pConn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("UDP forward %s read error: %v", direction, err)
				return
			}

			if n > 0 {
				// Forward to local service
				serviceConn.SetWriteDeadline(time.Now().Add(1 * time.Second))
				_, err = serviceConn.Write(buffer[:n])
				if err != nil {
					log.Printf("UDP forward %s write error: %v", direction, err)
					return
				}
			}
		}
	}()

	// Read responses from local service and send back to P2P
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		serviceConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := serviceConn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("UDP forward service->p2p read error: %v", err)
			return
		}

		if n > 0 {
			p2pConn.SetWriteDeadline(time.Now().Add(1 * time.Second))
			_, err = p2pConn.Write(buffer[:n])
			if err != nil {
				log.Printf("UDP forward service->p2p write error: %v", err)
				return
			}
		}
	}
}

// runUDPServerOnPort runs UDP server on specified port, forwarding to local service
func runUDPServerOnPort(ctx context.Context, listenPort, localServicePort int) {
	localPeerAddr := net.UDPAddr{Port: listenPort}
	conn, err := net.ListenUDP("udp", &localPeerAddr)
	if err != nil {
		log.Fatalf("UDP server listen error on port %d: %v", listenPort, err)
	}
	defer conn.Close()

	localServiceAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: localServicePort}
	buf := make([]byte, UDPBufferSize)

	log.Printf("UDP Server listening on port %d, forwarding to local service 127.0.0.1:%d", listenPort, localServicePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, peerAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP server read error: %v", err)
			continue
		}

		// Forward to local service
		go func(data []byte, peer *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &localServiceAddr)
			if err != nil {
				log.Printf("UDP server write to local service error: %v", err)
			}
		}(buf[:n], peerAddr)
	}
}