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

func tcpReceiver(localPort int, remoteIP string, remotePort int) {
	peer, err := net.Listen("tcp", ":"+strconv.Itoa(remotePort))
	if err != nil {
		log.Fatalf("tcpReceiver listen error: %v", err)
	}
	for {
		conn, err := peer.Accept()
		if err != nil {
			log.Printf("tcpReceiver accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			local, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(localPort))
			if err != nil {
				log.Printf("tcpReceiver dial error: %v", err)
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

func udpReceiver(localPort int, remoteIP string, remotePort int) {
	remoteAddr := net.UDPAddr{IP: net.ParseIP(remoteIP), Port: remotePort}
	conn, err := net.ListenUDP("udp", &remoteAddr)
	if err != nil {
		log.Fatalf("udpReceiver listen error: %v", err)
	}
	localAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: localPort}
	buf := make([]byte, 2048)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err == nil {
			conn.WriteToUDP(buf[:n], &localAddr)
		}
	}
}
