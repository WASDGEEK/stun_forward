// tcp_udp.go
package main

import (
	"io"
	"log"
	"net"
	"strconv"
)

func tcpSender(localPort int, remoteIP string, remotePort int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(localPort))
	if err != nil {
		log.Fatalf("tcpSender listen error: %v", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("tcpSender accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			peer, err := net.Dial("tcp", net.JoinHostPort(remoteIP, strconv.Itoa(remotePort)))
			if err != nil {
				log.Printf("tcpSender dial error: %v", err)
				return
			}
			go io.Copy(peer, c)
			go io.Copy(c, peer)
		}(conn)
	}
}

func tcpReceiver(m PortMap, peerHost string, peerPort int) {
	// Receiver listens on its RemotePort for connections from the sender
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.RemotePort))
	if err != nil {
		log.Fatalf("tcpReceiver listen error: %v", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("tcpReceiver accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			// Receiver connects to the local service on LocalPort
			local, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(m.LocalPort)))
			if err != nil {
				log.Printf("tcpReceiver dial local service error: %v", err)
				return
			}
			go io.Copy(local, c)
			go io.Copy(c, local)
		}(conn)
	}
}

func udpSender(localPort int, remoteIP string, remotePort int) {
	localAddr := net.UDPAddr{Port: localPort}
	conn, err := net.ListenUDP("udp", &localAddr)
	if err != nil {
		log.Fatalf("udpSender listen error: %v", err)
	}
	remoteAddr := net.UDPAddr{IP: net.ParseIP(remoteIP), Port: remotePort}
	buf := make([]byte, 2048)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err == nil {
			conn.WriteToUDP(buf[:n], &remoteAddr)
		}
	}
}

func udpReceiver(m PortMap, peerHost string, peerPort int) {
	// Receiver listens on its RemotePort for packets from the sender
	localPeerAddr := net.UDPAddr{Port: m.RemotePort}
	conn, err := net.ListenUDP("udp", &localPeerAddr)
	if err != nil {
		log.Fatalf("udpReceiver listen error: %v", err)
	}

	// Address of the local service to forward to
	localServiceAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: m.LocalPort}

	buf := make([]byte, 2048)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err == nil {
			// Forward received packet to local service
			conn.WriteToUDP(buf[:n], &localServiceAddr)
		}
	}
}