// main.go
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

func main() {
	configPath := flag.String("config", "", "Path to the configuration file.")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("The --config flag is required. Please provide the path to your config.json.")
	}

	// Read the configuration file
	configFile, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config Configuration
	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Validate configuration
	if config.Mode != "sender" && config.Mode != "receiver" {
		log.Fatal("Config error: 'mode' must be 'sender' or 'receiver'")
	}
	if config.SignalingURL == "" {
		log.Fatal("Config error: 'signalingUrl' is required")
	}
	if config.RoomID == "" {
		log.Fatal("Config error: 'roomId' is required")
	}
	if len(config.Mappings) == 0 {
		log.Fatal("Config error: at least one port 'mapping' is required")
	}
	if config.STUNServer == "" {
		// Provide a default STUN server if not specified
		config.STUNServer = "stun.l.google.com:19302"
	}

	runForwarder(config)
}