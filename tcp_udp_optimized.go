// tcp_udp_optimized.go - 优化版本的数据转发
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
	// OptimizedTCPBufferSize TCP传输的优化缓冲区大小
	OptimizedTCPBufferSize = 64 * 1024 // 64KB
	// OptimizedUDPBufferSize UDP传输的优化缓冲区大小  
	OptimizedUDPBufferSize = 8 * 1024 // 8KB
)

// optimizedTCPProxy 优化版的TCP代理，使用更大的缓冲区和更好的错误处理
func optimizedTCPProxy(ctx context.Context, src, dst net.Conn, direction string) {
	defer src.Close()
	defer dst.Close()

	buf := make([]byte, OptimizedTCPBufferSize)
	
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

// tcpSenderOptimized 优化版的TCP发送端
func tcpSenderOptimized(ctx context.Context, localPort int, remoteIP string, remotePort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(localPort))
	if err != nil {
		log.Fatalf("tcpSenderOptimized listen error: %v", err)
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
			log.Printf("tcpSenderOptimized accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			
			peer, err := net.Dial("tcp", net.JoinHostPort(remoteIP, strconv.Itoa(remotePort)))
			if err != nil {
				log.Printf("tcpSenderOptimized dial error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// 客户端到peer
			go func() {
				defer wg.Done()
				optimizedTCPProxy(ctx, c, peer, "client->peer")
			}()

			// peer到客户端
			go func() {
				defer wg.Done() 
				optimizedTCPProxy(ctx, peer, c, "peer->client")
			}()

			wg.Wait()
		}(conn)
	}
}

// tcpReceiverOptimized 优化版的TCP接收端
func tcpReceiverOptimized(ctx context.Context, m PortMap, peerHost string, peerPort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.RemotePort))
	if err != nil {
		log.Fatalf("tcpReceiverOptimized listen error: %v", err)
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
			log.Printf("tcpReceiverOptimized accept error: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			local, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(m.LocalPort)))
			if err != nil {
				log.Printf("tcpReceiverOptimized dial local service error: %v", err)
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// peer到本地服务
			go func() {
				defer wg.Done()
				optimizedTCPProxy(ctx, c, local, "peer->local")
			}()

			// 本地服务到peer
			go func() {
				defer wg.Done()
				optimizedTCPProxy(ctx, local, c, "local->peer")
			}()

			wg.Wait()
		}(conn)
	}
}

// udpSenderOptimized 优化版的UDP发送端
func udpSenderOptimized(ctx context.Context, localPort int, remoteIP string, remotePort int) {
	localAddr := net.UDPAddr{Port: localPort}
	conn, err := net.ListenUDP("udp", &localAddr)
	if err != nil {
		log.Fatalf("udpSenderOptimized listen error: %v", err)
	}
	defer conn.Close()

	remoteAddr := net.UDPAddr{IP: net.ParseIP(remoteIP), Port: remotePort}
	buf := make([]byte, OptimizedUDPBufferSize)
	
	log.Printf("UDP Sender listening on port %d, forwarding to %s:%d", localPort, remoteIP, remotePort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("udpSenderOptimized read error: %v", err)
			continue
		}

		// 转发到远程peer
		go func(data []byte, client *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &remoteAddr)
			if err != nil {
				log.Printf("udpSenderOptimized write to remote error: %v", err)
			}
		}(buf[:n], clientAddr)
	}
}

// udpReceiverOptimized 优化版的UDP接收端
func udpReceiverOptimized(ctx context.Context, m PortMap, peerHost string, peerPort int) {
	localPeerAddr := net.UDPAddr{Port: m.RemotePort}
	conn, err := net.ListenUDP("udp", &localPeerAddr)
	if err != nil {
		log.Fatalf("udpReceiverOptimized listen error: %v", err)
	}
	defer conn.Close()

	localServiceAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: m.LocalPort}
	buf := make([]byte, OptimizedUDPBufferSize)

	log.Printf("UDP Receiver listening on port %d, forwarding to local service 127.0.0.1:%d", m.RemotePort, m.LocalPort)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, peerAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("udpReceiverOptimized read error: %v", err)
			continue
		}

		// 转发到本地服务
		go func(data []byte, peer *net.UDPAddr) {
			_, err := conn.WriteToUDP(data, &localServiceAddr)
			if err != nil {
				log.Printf("udpReceiverOptimized write to local service error: %v", err)
			}
		}(buf[:n], peerAddr)
	}
}