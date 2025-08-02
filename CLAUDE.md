# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an advanced Go-based P2P NAT traversal tool with **true UDP hole punching** and **dynamic mapping management**. It creates intelligent tunnels between clients and servers behind NATs using enhanced STUN discovery, comprehensive NAT type detection, and an enhanced PHP signaling server for real-time coordination. The system enables automatic port forwarding without manual router configuration while supporting hot configuration updates.

## Build and Development Commands

### Build
```bash
go build -o stun_forward .
```

### Cross-platform builds (as used in CI)
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o stun_forward-linux-amd64 .

# Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o stun_forward-windows-amd64.exe .

# macOS ARM64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o stun_forward-darwin-arm64 .

# Android ARM64 (requires Android NDK)
GOOS=android GOARCH=arm64 CGO_ENABLED=1 go build -o stun_forward-android-arm64 .
```

### Run
```bash
./stun_forward --config /path/to/config.yml
# Or use default config.yml:
./stun_forward
```

### Test
```bash
go test ./...
# Note: Currently no test files exist in the codebase
```

### Dependencies
```bash
go mod tidy
go mod download
```

### Format and Lint
```bash
go fmt ./...
go vet ./...
```

## Architecture

### Core Components

- **main.go**: Entry point, configuration validation, CLI argument parsing, supports both YAML and JSON configs
- **types.go**: Enhanced data structures (`Configuration`, `PortMapping`, `NetworkInfo`, `STUNResult`) with flexible JSON/YAML unmarshaling
- **run.go**: Advanced execution logic with client/server modes, enhanced LAN detection, dynamic mapping updates, and concurrent port management
- **stun.go**: Comprehensive STUN implementation with NAT type detection (Full Cone, Restricted Cone, Port Restricted, Symmetric NAT)
- **holepunch.go**: Advanced UDP hole punching with simultaneous connect, port prediction, and multi-strategy fallback
- **signaling.go**: Enhanced HTTP client with mapping update support, version control, and real-time synchronization
- **forwarder.go**: Protocol-specific forwarding with P2P hole punching integration and relay fallback
- **mapping_updater.go**: Dynamic mapping management with interactive CLI and hot configuration updates
- **signaling/signaling_server_enhanced.php**: Advanced PHP signaling server with auto-cleanup, version control, and real-time updates

### Enhanced Client/Server Model with P2P and Dynamic Management

- **Client Mode**: Defines and manages port mappings dynamically, performs NAT detection, establishes P2P or relay connections
- **Server Mode**: Dynamically allocates ports, performs NAT detection, supports real-time mapping updates, manages P2P hole punching
- **Smart Connection Selection**: Multi-strategy approach (LAN Direct → UDP Hole Punch → TCP/UDP Relay)
- **Real-time Coordination**: Enhanced signaling server with version control, auto-cleanup, and live mapping synchronization
- **Interactive Management**: CLI interface for dynamic mapping updates without service interruption

### Enhanced Data Flow

**Initial Connection:**
1. Both client/server perform comprehensive NAT type detection and network discovery
2. Client posts enhanced network info + mapping requirements to signaling server with version control
3. Server retrieves client requirements and dynamically allocates available ports
4. Server posts network info + port allocation results with mapping version tracking
5. Connection establishment using optimal method (LAN Direct/UDP Hole Punch/TCP Relay)

**Dynamic Updates:**
6. Server continuously monitors for mapping updates via enhanced signaling protocol
7. Client can modify mappings through interactive CLI or configuration changes
8. Server automatically reallocates ports and establishes new connections without interruption
9. Enhanced signaling server provides auto-cleanup (5-minute inactivity timeout)

### Key Improvements

- **Advanced NAT Traversal**: True UDP hole punching with multi-strategy fallback (simultaneous connect, port prediction)
- **Real-time Dynamic Updates**: Hot mapping management without service restart
- **Enhanced Signaling Protocol**: Version control, auto-cleanup, and conflict resolution
- **Comprehensive NAT Detection**: Full Cone, Restricted Cone, Port Restricted, Symmetric NAT identification
- **Smart Connection Optimization**: Automatic LAN detection and P2P optimization
- **Interactive Management**: CLI-based mapping updates with real-time feedback
- **Resource Management**: Automatic room cleanup and memory leak prevention

### Configuration

The tool supports both YAML (.yml/.yaml) and JSON (.json) configuration files with enhanced fields:
- `mode`: "client" or "server"
- `roomId`: Shared secret for peer matching (use cryptographically secure strings)
- `signalingUrl`: Enhanced PHP signaling server endpoint (`signaling_server_enhanced.php` recommended)
- `stunServer`: STUN server for NAT traversal (optional, defaults to Google's STUN)
- `mappings`: Array of "protocol:localPort:remotePort" strings (client only, supports hot updates)

**Example Enhanced Configuration:**
```yaml
mode: client
roomId: "secure-random-room-id"
signalingUrl: "https://your-server.com/signaling_server_enhanced.php"
stunServer: "stun.l.google.com:19302"
mappings:
  - "udp:5000:53"    # DNS with UDP hole punching
  - "tcp:8080:80"    # HTTP with TCP relay
```

### LAN Detection

The tool implements multi-strategy LAN detection:
- Same public IP detection (most reliable for NAT scenarios)
- Private IP subnet analysis across standard ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
- Automatic fallback to WAN connection when LAN detection fails

## Key Development Notes

### Architecture Changes
- Client uses `handleClientMode` function (run.go:37) for centralized registration
- Server uses `handleServerMode` function (run.go:40) with dynamic port allocation
- Each port mapping runs in its own goroutine (`handlePortMappingWithAllocatedPort` function)
- Server allocates ports using `allocatePortForMapping` function

### Data Structures
- `ClientRegistrationData`: Contains network info + mapping requirements
- `ServerRegistrationData`: Contains network info + port allocation results
- `ServerPortMapping`: Maps client requirements to allocated ports

### Key Functions

**Core Network Functions:**
- `discoverNATType` (stun.go): Comprehensive NAT type detection with multiple STUN servers
- `performUDPHolePunching` (holepunch.go): Multi-strategy P2P connection establishment
- `establishP2PConnection` (holepunch.go): High-level P2P connection with fallback
- `runUDPClientWithHolePunching`/`runUDPServerWithHolePunching` (forwarder.go): P2P-enabled data forwarding

**Enhanced Management Functions:**
- `handleClientMode` (run.go): Enhanced client with NAT detection and mapping management
- `handleServerMode` (run.go): Advanced server with real-time mapping updates monitoring
- `handleMappingUpdate` (run.go): Dynamic mapping update processing with port reallocation
- `NewMappingUpdater` (mapping_updater.go): Interactive CLI for real-time mapping changes

**Signaling Protocol Functions:**
- `UpdateMappings` (signaling.go): Send mapping updates to enhanced signaling server
- `WatchMappingUpdates` (signaling.go): Continuous monitoring for mapping changes
- `CheckMappingUpdates` (signaling.go): Version-controlled update detection

**Advanced Features:**
- Enhanced configuration parsing with flexible JSON/YAML unmarshaling
- Multi-strategy network discovery (STUN + local interface detection + NAT type analysis)
- Smart connection optimization (LAN Direct → UDP Hole Punch → TCP/UDP Relay)
- Resource management with automatic cleanup and memory leak prevention

### Enhanced Port Allocation System
- **Dynamic Allocation**: Server uses system port allocation (`:0`) to avoid conflicts
- **Real-time Updates**: Supports hot reallocation when mappings change
- **Multi-protocol Support**: Each client mapping gets optimized connection method
- **P2P Integration**: Port allocation coordinates with hole punching requirements
- **Version Control**: Port allocation tracking with conflict resolution
- **Concurrent Safety**: Thread-safe port management for multiple simultaneous mappings
- **Resource Efficiency**: Automatic port cleanup when mappings are removed

### Enhanced Debugging and Troubleshooting

**NAT Detection Logging:**
- Detailed NAT type detection with multiple server testing
- Connection method selection reasoning (LAN/P2P/Relay)
- Hole punching attempt details and fallback triggers

**Real-time Monitoring:**
- Mapping update detection and processing logs
- Version control and conflict resolution tracking  
- Port allocation and deallocation lifecycle logging

**Performance Analytics:**
- Connection establishment timing and success rates
- P2P vs relay usage statistics
- Network optimization decision reasoning

**Enhanced Debug Messages:**
```
🔍 NAT Detection - NAT Type: Full Cone NAT
🎯 Using UDP hole punching for port 5000
🔄 Detected mapping updates from client
✅ Successfully processed mapping update - 3 new port allocations
👀 Starting mapping updates watcher for room: my-room
```

**Troubleshooting Tools:**
- Interactive CLI with `help` command for guided troubleshooting
- Detailed error messages with suggested remediation steps
- Connection diagnostics with performance recommendations

## CI/CD Pipeline

The project uses GitHub Actions for automated builds:
- **go.yml**: Cross-platform builds for Linux, Windows, macOS (ARM64/AMD64)
- **go-android-arm64.yml**: Specialized Android ARM64 builds with NDK
- All builds use Go 1.24.4 and produce compressed artifacts
- CGO disabled for most platforms except Android

## Development Notes

### Code Organization
- Single-package structure (`main`) with focused file responsibilities
- No external test files currently exist (opportunity for improvement)
- Configuration supports both YAML and JSON via flexible unmarshaling
- Extensive logging and debugging output for network operations

### Key Dependencies
- `github.com/pion/stun`: STUN protocol implementation
- `gopkg.in/yaml.v3`: YAML configuration parsing
- Standard Go networking libraries for UDP/TCP operations

### Current Project Status
- Active development with recent P2P and dynamic mapping features
- Production-ready with enhanced signaling server
- Multi-platform support including mobile (Android ARM64)
- No formal test suite (manual testing workflow)