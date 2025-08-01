// main.go
package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	mode := flag.String("mode", "", "sender, receiver, or server")
	room := flag.String("room", "", "Room ID to match peers")
	signalURL := flag.String("signal", "", "Signal server URL")
	maps := flag.String("map", "", "Comma-separated port mappings. Format: proto:local:remote")
	port := flag.String("port", "8080", "Port for the signal server to listen on")
	flag.Parse()

	if *mode == "server" {
		runServer(*port)
		return
	}

	if *mode != "sender" && *mode != "receiver" {
		log.Fatal("--mode must be sender or receiver")
	}
	if *signalURL == "" {
		log.Fatal("--signal is required")
	}
	if *maps == "" {
		log.Fatal("--map is required")
	}
	if *room == "" {
		log.Fatal("--room is required")
	}

	config := Config{
		Mode:      *mode,
		Room:      *room,
		SignalURL: *signalURL,
		Mappings:  parseMappings(*maps),
	}

	Run(config)
}

func parseMappings(raw string) []PortMap {
	var result []PortMap
	items := strings.Split(raw, ",")
	for _, item := range items {
		pm, err := ParsePortMap(item)
		if err != nil {
			log.Fatalf("Invalid --map value: %v", err)
		}
		result = append(result, pm)
	}
	return result
}