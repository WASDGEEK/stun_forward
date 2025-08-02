// Package main - Dynamic mapping update functionality
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// MappingUpdater handles dynamic mapping updates for client
type MappingUpdater struct {
	config          Configuration
	signalingClient *SignalingClient
	roomKey         string
	currentMappings []PortMapping
}

// NewMappingUpdater creates a new mapping updater
func NewMappingUpdater(config Configuration, signalingClient *SignalingClient, roomKey string, initialMappings []PortMapping) *MappingUpdater {
	return &MappingUpdater{
		config:          config,
		signalingClient: signalingClient,
		roomKey:         roomKey,
		currentMappings: initialMappings,
	}
}

// StartInteractiveUpdater starts an interactive CLI for mapping updates
func (mu *MappingUpdater) StartInteractiveUpdater(ctx context.Context) {
	log.Printf("🎛️  Interactive mapping updater started")
	log.Printf("Commands:")
	log.Printf("  add <protocol:localPort:remotePort> - Add new mapping")
	log.Printf("  remove <index> - Remove mapping by index")
	log.Printf("  list - Show current mappings")
	log.Printf("  update - Send current mappings to server")
	log.Printf("  help - Show this help")
	log.Printf("  quit - Exit updater")
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		fmt.Print("mapping> ")
		if !scanner.Scan() {
			return
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}
		
		command := strings.ToLower(parts[0])
		
		switch command {
		case "add":
			if len(parts) != 2 {
				fmt.Println("Usage: add <protocol:localPort:remotePort>")
				continue
			}
			mu.addMapping(parts[1])
			
		case "remove":
			if len(parts) != 2 {
				fmt.Println("Usage: remove <index>")
				continue
			}
			mu.removeMapping(parts[1])
			
		case "list":
			mu.listMappings()
			
		case "update":
			mu.sendMappingUpdate()
			
		case "help":
			fmt.Println("Commands:")
			fmt.Println("  add <protocol:localPort:remotePort> - Add new mapping")
			fmt.Println("  remove <index> - Remove mapping by index")
			fmt.Println("  list - Show current mappings")
			fmt.Println("  update - Send current mappings to server")
			fmt.Println("  help - Show this help")
			fmt.Println("  quit - Exit updater")
			
		case "quit", "exit":
			log.Printf("Exiting mapping updater...")
			return
			
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}
}

// addMapping adds a new mapping
func (mu *MappingUpdater) addMapping(mappingStr string) {
	var mapping PortMapping
	err := mapping.parseFromString(mappingStr)
	if err != nil {
		fmt.Printf("❌ Invalid mapping format: %v\n", err)
		return
	}
	
	// Check for duplicates
	for _, existing := range mu.currentMappings {
		if existing.Protocol == mapping.Protocol && existing.LocalPort == mapping.LocalPort {
			fmt.Printf("❌ Mapping with same protocol and local port already exists\n")
			return
		}
	}
	
	mu.currentMappings = append(mu.currentMappings, mapping)
	fmt.Printf("✅ Added mapping: %s %d->%d\n", mapping.Protocol, mapping.LocalPort, mapping.RemotePort)
}

// removeMapping removes a mapping by index
func (mu *MappingUpdater) removeMapping(indexStr string) {
	var index int
	_, err := fmt.Sscanf(indexStr, "%d", &index)
	if err != nil {
		fmt.Printf("❌ Invalid index: %s\n", indexStr)
		return
	}
	
	if index < 0 || index >= len(mu.currentMappings) {
		fmt.Printf("❌ Index out of range: %d (valid range: 0-%d)\n", index, len(mu.currentMappings)-1)
		return
	}
	
	removed := mu.currentMappings[index]
	mu.currentMappings = append(mu.currentMappings[:index], mu.currentMappings[index+1:]...)
	fmt.Printf("✅ Removed mapping: %s %d->%d\n", removed.Protocol, removed.LocalPort, removed.RemotePort)
}

// listMappings shows current mappings
func (mu *MappingUpdater) listMappings() {
	if len(mu.currentMappings) == 0 {
		fmt.Println("📝 No mappings configured")
		return
	}
	
	fmt.Printf("📝 Current mappings (%d):\n", len(mu.currentMappings))
	for i, mapping := range mu.currentMappings {
		fmt.Printf("  [%d] %s %d->%d\n", i, mapping.Protocol, mapping.LocalPort, mapping.RemotePort)
	}
}

// sendMappingUpdate sends current mappings to server
func (mu *MappingUpdater) sendMappingUpdate() {
	fmt.Printf("📤 Sending %d mappings to server...\n", len(mu.currentMappings))
	
	// Convert mappings to string format
	var mappingStrings []string
	for _, mapping := range mu.currentMappings {
		mappingStr := fmt.Sprintf("%s:%d:%d", mapping.Protocol, mapping.LocalPort, mapping.RemotePort)
		mappingStrings = append(mappingStrings, mappingStr)
	}
	
	err := mu.signalingClient.UpdateMappings(mu.config.SignalingURL, mu.roomKey, mappingStrings)
	if err != nil {
		fmt.Printf("❌ Failed to send mapping update: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Mapping update sent successfully\n")
	
	// Wait a moment for server to process and then check for new allocations
	time.Sleep(2 * time.Second)
	
	serverData, err := mu.signalingClient.WaitForPeerData(context.Background(), mu.config.SignalingURL, 
		peerRole(mu.config.Mode), mu.roomKey, 5*time.Second)
	if err != nil {
		fmt.Printf("⚠️  Could not retrieve updated server data: %v\n", err)
		return
	}
	
	serverRegistration, err := parseServerRegistrationData(serverData)
	if err != nil {
		fmt.Printf("⚠️  Could not parse updated server data: %v\n", err)
		return
	}
	
	fmt.Printf("🎯 Server allocated new ports:\n")
	for _, portMapping := range serverRegistration.PortMappings {
		mapping := portMapping.ClientMapping
		fmt.Printf("  %s %d->%d allocated port: %d\n", 
			mapping.Protocol, mapping.LocalPort, mapping.RemotePort, portMapping.AllocatedPort)
	}
}

// AutoUpdateFromConfig automatically updates mappings from config file changes
func (mu *MappingUpdater) AutoUpdateFromConfig(ctx context.Context, configPath string) {
	log.Printf("👀 Starting config file watcher for: %s", configPath)
	
	lastModTime := time.Time{}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat, err := os.Stat(configPath)
			if err != nil {
				continue
			}
			
			if stat.ModTime().After(lastModTime) {
				lastModTime = stat.ModTime()
				
				// Skip first iteration (initial load)
				if lastModTime.IsZero() {
					continue
				}
				
				log.Printf("📄 Config file changed, reloading mappings...")
				
				newConfig, err := parseConfig(configPath)
				if err != nil {
					log.Printf("❌ Failed to reload config: %v", err)
					continue
				}
				
				// Check if mappings actually changed
				if mappingsEqual(mu.currentMappings, newConfig.Mappings) {
					continue
				}
				
				mu.currentMappings = newConfig.Mappings
				log.Printf("🔄 Detected %d mapping changes, updating server...", len(mu.currentMappings))
				
				mu.sendMappingUpdate()
			}
		}
	}
}

// mappingsEqual compares two mapping slices for equality
func mappingsEqual(a, b []PortMapping) bool {
	if len(a) != len(b) {
		return false
	}
	
	for i := range a {
		if a[i].Protocol != b[i].Protocol || 
		   a[i].LocalPort != b[i].LocalPort || 
		   a[i].RemotePort != b[i].RemotePort {
			return false
		}
	}
	
	return true
}