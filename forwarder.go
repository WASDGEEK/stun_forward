// Package main - Network forwarding implementations
package main

import (
	"context"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
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