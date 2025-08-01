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

// runTCPSender runs TCP sender forwarding
func runTCPSender(ctx context.Context, localPort int, remoteIP string, remotePort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(localPort))
	if err != nil {
		log.Fatalf("TCP sender listen error: %v", err)
	}
	defer ln.Close()

	log.Printf("TCP Sender listening on port %d, forwarding to %s:%d", localPort, remoteIP, remotePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			log.Printf("TCP sender accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			
			peer, err := net.Dial("tcp", net.JoinHostPort(remoteIP, strconv.Itoa(remotePort)))
			if err != nil {
				log.Printf("TCP sender dial error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// Client to peer
			go func() {
				defer wg.Done()
				tcpProxy(ctx, c, peer, "client->peer")
			}()

			// Peer to client
			go func() {
				defer wg.Done() 
				tcpProxy(ctx, peer, c, "peer->client")
			}()

			wg.Wait()
		}(conn)
	}
}

// runTCPReceiver runs TCP receiver forwarding
func runTCPReceiver(ctx context.Context, m PortMapping, peerHost string, peerPort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.RemotePort))
	if err != nil {
		log.Fatalf("TCP receiver listen error: %v", err)
	}
	defer ln.Close()

	log.Printf("TCP Receiver listening on port %d, forwarding to local service 127.0.0.1:%d", m.RemotePort, m.LocalPort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			log.Printf("TCP receiver accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			local, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(m.LocalPort)))
			if err != nil {
				log.Printf("TCP receiver dial local service error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// Peer to local service
			go func() {
				defer wg.Done()
				tcpProxy(ctx, c, local, "peer->local")
			}()

			// Local service to peer
			go func() {
				defer wg.Done()
				tcpProxy(ctx, local, c, "local->peer")
			}()

			wg.Wait()
		}(conn)
	}
}

// runUDPSender runs UDP sender forwarding
func runUDPSender(ctx context.Context, localPort int, remoteIP string, remotePort int) {
	localAddr := net.UDPAddr{Port: localPort}
	conn, err := net.ListenUDP("udp", &localAddr)
	if err != nil {
		log.Fatalf("UDP sender listen error: %v", err)
	}
	defer conn.Close()

	remoteAddr := net.UDPAddr{IP: net.ParseIP(remoteIP), Port: remotePort}
	buf := make([]byte, UDPBufferSize)
	
	log.Printf("UDP Sender listening on port %d, forwarding to %s:%d", localPort, remoteIP, remotePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP sender read error: %v", err)
			continue
		}

		// Forward to remote peer
		go func(data []byte, client *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &remoteAddr)
			if err != nil {
				log.Printf("UDP sender write to remote error: %v", err)
			}
		}(buf[:n], clientAddr)
	}
}

// runUDPReceiver runs UDP receiver forwarding
func runUDPReceiver(ctx context.Context, m PortMapping, peerHost string, peerPort int) {
	localPeerAddr := net.UDPAddr{Port: m.RemotePort}
	conn, err := net.ListenUDP("udp", &localPeerAddr)
	if err != nil {
		log.Fatalf("UDP receiver listen error: %v", err)
	}
	defer conn.Close()

	localServiceAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: m.LocalPort}
	buf := make([]byte, UDPBufferSize)

	log.Printf("UDP Receiver listening on port %d, forwarding to local service 127.0.0.1:%d", m.RemotePort, m.LocalPort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, peerAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP receiver read error: %v", err)
			continue
		}

		// Forward to local service
		go func(data []byte, peer *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &localServiceAddr)
			if err != nil {
				log.Printf("UDP receiver write to local service error: %v", err)
			}
		}(buf[:n], peerAddr)
	}
}