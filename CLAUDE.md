# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based P2P NAT traversal tool that creates tunnels between a client and server behind NATs using STUN for discovery and a PHP signaling server for coordination. It enables port forwarding without manual router configuration.

## Build and Development Commands

### Build
```bash
go build -o stun_forward .
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
```

### Dependencies
```bash
go mod tidy
go mod download
```

## Architecture

### Core Components

- **main.go**: Entry point, configuration validation, CLI argument parsing, supports both YAML and JSON configs
- **types.go**: Core data structures (`Configuration`, `PortMapping`) with custom JSON/YAML unmarshaling
- **run.go**: Main execution logic with client/server modes, LAN detection, and concurrent port mapping
- **stun.go**: STUN client implementation for public IP discovery using github.com/pion/stun
- **signaling.go**: HTTP client for signaling server communication (peer coordination)
- **forwarder.go**: Protocol-specific TCP/UDP forwarding implementations
- **index.php**: Simple PHP signaling server for peer discovery and coordination

### Client/Server Model with Dynamic Port Allocation

- **Client Mode**: Defines port mappings, sends requirements to server, listens on local ports
- **Server Mode**: Dynamically allocates ports based on client requirements, forwards to local services
- **Smart Routing**: Automatically detects LAN connections and uses direct private IP when possible
- **Port Coordination**: Uses signaling server to exchange port allocation information

### Data Flow

1. Both client/server discover public and private IPs via STUN and local network detection
2. Client posts network info + mapping requirements to signaling server
3. Server retrieves client requirements and dynamically allocates available ports
4. Server posts network info + port allocation results to signaling server
5. Client retrieves server's port allocations and connects to allocated ports
6. Server forwards traffic from allocated ports to local services

### Configuration

The tool supports both YAML (.yml/.yaml) and JSON (.json) configuration files with these key fields:
- `mode`: "client" or "server"
- `roomId`: Shared secret for peer matching  
- `signalingUrl`: PHP signaling server endpoint
- `stunServer`: STUN server for NAT traversal (optional, defaults to Google's)
- `mappings`: Array of "protocol:localPort:remotePort" strings (client only)

### LAN Detection

The tool implements multi-strategy LAN detection:
- Same public IP detection (most reliable for NAT scenarios)
- Private IP subnet analysis across standard ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
- Automatic fallback to WAN connection when LAN detection fails

## Key Development Notes

### Architecture Changes
- Client uses `handleClientMode` function (run.go:52) for centralized registration
- Server uses `handleServerMode` function (run.go:196) with dynamic port allocation
- Each port mapping runs in its own goroutine (`handlePortMappingWithAllocatedPort` function in run.go:113)
- Server allocates ports using `allocatePortForMapping` function (run.go:165)

### Data Structures
- `ClientRegistrationData`: Contains network info + mapping requirements
- `ServerRegistrationData`: Contains network info + port allocation results
- `ServerPortMapping`: Maps client requirements to allocated ports

### Key Functions
- `runTCPServerOnPort`/`runUDPServerOnPort` (forwarder.go): Listen on allocated ports, forward to local services
- `formatClientRegistrationData`/`parseServerRegistrationData` (run.go): Handle JSON serialization
- Configuration parsing supports flexible string-to-struct conversion via custom UnmarshalJSON/UnmarshalYAML
- Network discovery combines STUN (public) and local interface detection (private)
- LAN optimization bypasses STUN when peers are detected on same network
- Graceful shutdown handling with context cancellation and signal handling

### Port Allocation System
- Server uses system port allocation (`:0`) to avoid conflicts
- Each client mapping gets a unique server port
- Port allocation info exchanged via signaling server
- Supports concurrent multiple mappings without conflicts