// run.go
package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func peer(mode string) string {
	if mode == "sender" {
		return "receiver"
	}
	return "sender"
}

func Run(cfg Config) {
	for _, m := range cfg.Mappings {
		go handleMapping(cfg, m)
	}
	select {} // block forever
}

func handleMapping(cfg Config, m PortMap) {
	log.Printf("[%s] Preparing port forward: %s %d <-> %d", cfg.Mode, m.Proto, m.LocalPort, m.RemotePort)

	localAddr := getLocalAddress()
	myInfo := fmt.Sprintf("%s:%d", localAddr, m.RemotePort)

	err := PostSignal(cfg.SignalURL, cfg.Mode, cfg.Room+"-"+protoPortKey(m), myInfo)
	if err != nil {
		log.Fatalf("signal post failed: %v", err)
	}

	peerInfo, err := WaitForPeerData(cfg.SignalURL, peer(cfg.Mode), cfg.Room+"-"+protoPortKey(m), 30*time.Second)
	if err != nil {
		log.Fatalf("waiting for peer failed: %v", err)
	}

	host, portStr, _ := strings.Cut(peerInfo, ":")
	port, _ := strconv.Atoi(portStr)

	if cfg.Mode == "sender" {
		// sender connects to peer, receives from local
		if m.Proto == "tcp" {
			tcpSender(m.LocalPort, host, port)
		} else {
			udpSender(m.LocalPort, host, port)
		}
	} else {
		// receiver listens on local port, connects to peer
		if m.Proto == "tcp" {
			tcpReceiver(m.LocalPort, host, port)
		} else {
			udpReceiver(m.LocalPort, host, port)
		}
	}
}

func protoPortKey(m PortMap) string {
	return fmt.Sprintf("%s-%d-%d", m.Proto, m.LocalPort, m.RemotePort)
}

func getLocalAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}
