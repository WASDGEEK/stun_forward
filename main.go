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

	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Validate configuration
	if config.Mode != "sender" && config.Mode != "receiver" {
		log.Fatal("Config error: 'mode' must be 'sender' or 'receiver'")
	}
	if config.SignalURL == "" {
		log.Fatal("Config error: 'signalURL' is required")
	}
	if config.Room == "" {
		log.Fatal("Config error: 'room' is required")
	}
	if len(config.Mappings) == 0 {
		log.Fatal("Config error: at least one port 'mapping' is required")
	}
	if config.StunServer == "" {
		// Provide a default STUN server if not specified
		config.StunServer = "stun.l.google.com:19302"
	}

	Run(config)
}