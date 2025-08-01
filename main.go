// main.go
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", "config.yml", "Path to the configuration file (default: config.yml)")
	flag.Parse()

	// Use default config.yml if no config specified and it exists
	if *configPath == "config.yml" {
		if _, err := os.Stat("config.yml"); os.IsNotExist(err) {
			log.Fatal("No configuration file found. Please create config.yml or specify --config flag.")
		}
	}

	// Read the configuration file
	configFile, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config Configuration
	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(*configPath))
	switch ext {
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(configFile, &config); err != nil {
			log.Fatalf("Failed to parse YAML config file: %v", err)
		}
	case ".json":
		if err := json.Unmarshal(configFile, &config); err != nil {
			log.Fatalf("Failed to parse JSON config file: %v", err)
		}
	default:
		log.Fatalf("Unsupported config file format. Use .yml, .yaml, or .json")
	}

	// Validate configuration
	if config.Mode != "client" && config.Mode != "server" {
		log.Fatal("Config error: 'mode' must be 'client' or 'server'")
	}
	if config.SignalingURL == "" {
		log.Fatal("Config error: 'signalingUrl' is required")
	}
	if config.RoomID == "" {
		log.Fatal("Config error: 'roomId' is required")
	}
	// Only client needs mappings
	if config.Mode == "client" && len(config.Mappings) == 0 {
		log.Fatal("Config error: client mode requires at least one port 'mapping'")
	}
	// Server ignores mappings
	if config.Mode == "server" {
		config.Mappings = nil // Clear any mappings for server
	}
	if config.STUNServer == "" {
		// Provide a default STUN server if not specified
		config.STUNServer = "stun.l.google.com:19302"
	}

	runForwarder(config)
}