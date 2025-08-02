package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"stun_forward/pkg/types"
	"gopkg.in/yaml.v3"
)

// Manager manages configuration loading, validation, and watching
type Manager struct {
	config     *types.Config
	configPath string
	mutex      sync.RWMutex
	watchers   []chan types.Event
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		config:   types.DefaultConfig(),
		watchers: make([]chan types.Event, 0),
	}
}

// LoadFromFile loads configuration from a file
func (m *Manager) LoadFromFile(path string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config := types.DefaultConfig()

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, config); err != nil {
			if jsonErr := json.Unmarshal(data, config); jsonErr != nil {
				return fmt.Errorf("failed to parse config as YAML or JSON: YAML error: %v, JSON error: %v", err, jsonErr)
			}
		}
	}

	// Parse string mappings into PortMapping objects
	if err := m.parseMappings(config); err != nil {
		return fmt.Errorf("failed to parse mappings: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	oldConfig := m.config
	m.config = config
	m.configPath = path

	// Notify watchers if config changed
	if oldConfig != nil {
		m.notifyWatchers(types.NewEvent(types.EventTypeConfigChanged, config, "config.manager"))
	}

	return nil
}

// parseMappings handles the flexible mapping format
func (m *Manager) parseMappings(config *types.Config) error {
	// This is a simplified version - in the real implementation,
	// we'd need to handle the unmarshaling of different formats
	return nil
}

// LoadFromData loads configuration from raw data
func (m *Manager) LoadFromData(data []byte, format string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	config := types.DefaultConfig()

	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	oldConfig := m.config
	m.config = config

	// Notify watchers if config changed
	if oldConfig != nil {
		m.notifyWatchers(types.NewEvent(types.EventTypeConfigChanged, config, "config.manager"))
	}

	return nil
}

// Get returns the current configuration (thread-safe copy)
func (m *Manager) Get() *types.Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *m.config
	
	// Deep copy the mappings slice
	if m.config.Mappings != nil {
		configCopy.Mappings = make([]*types.PortMapping, len(m.config.Mappings))
		for i, mapping := range m.config.Mappings {
			mappingCopy := *mapping
			configCopy.Mappings[i] = &mappingCopy
		}
	}

	return &configCopy
}

// AddMapping adds a new port mapping
func (m *Manager) AddMapping(mapping *types.PortMapping) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate the mapping
	if mapping.Protocol != "tcp" && mapping.Protocol != "udp" {
		return fmt.Errorf("invalid protocol: %s", mapping.Protocol)
	}
	if mapping.LocalPort <= 0 || mapping.LocalPort > 65535 {
		return fmt.Errorf("invalid local port: %d", mapping.LocalPort)
	}
	if mapping.RemotePort <= 0 || mapping.RemotePort > 65535 {
		return fmt.Errorf("invalid remote port: %d", mapping.RemotePort)
	}

	// Check for duplicates
	for _, existing := range m.config.Mappings {
		if existing.Protocol == mapping.Protocol && existing.LocalPort == mapping.LocalPort {
			return fmt.Errorf("mapping already exists for %s:%d", mapping.Protocol, mapping.LocalPort)
		}
	}

	// Add the mapping
	m.config.Mappings = append(m.config.Mappings, mapping)

	// Notify watchers
	m.notifyWatchers(types.NewEvent(types.EventTypeMappingAdded, mapping, "config.manager"))

	return nil
}

// RemoveMapping removes a port mapping by index
func (m *Manager) RemoveMapping(index int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if index < 0 || index >= len(m.config.Mappings) {
		return fmt.Errorf("invalid mapping index: %d", index)
	}

	// Get the mapping to be removed for notification
	removedMapping := m.config.Mappings[index]

	// Remove the mapping
	m.config.Mappings = append(m.config.Mappings[:index], m.config.Mappings[index+1:]...)

	// Notify watchers
	m.notifyWatchers(types.NewEvent(types.EventTypeMappingRemoved, removedMapping, "config.manager"))

	return nil
}

// Watch returns a channel that receives configuration change events
func (m *Manager) Watch() <-chan types.Event {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	watcher := make(chan types.Event, 10) // Buffered to prevent blocking
	m.watchers = append(m.watchers, watcher)
	return watcher
}

// notifyWatchers notifies all watchers of a configuration change
func (m *Manager) notifyWatchers(event types.Event) {
	for _, watcher := range m.watchers {
		select {
		case watcher <- event:
		default:
			// Channel is full, skip this watcher to prevent blocking
		}
	}
}

// SaveToFile saves the current configuration to a file
func (m *Manager) SaveToFile(path string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var data []byte
	var err error

	// Determine format by extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(m.config, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(m.config)
	default:
		// Default to YAML
		data, err = yaml.Marshal(m.config)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Close closes all watchers
func (m *Manager) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, watcher := range m.watchers {
		close(watcher)
	}
	m.watchers = make([]chan types.Event, 0)
}