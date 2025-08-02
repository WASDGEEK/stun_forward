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

### Client/Server Model

- **Client Mode**: Actively connects to server, defines port mappings, listens on local ports
- **Server Mode**: Passively waits for connections, serves local services, no mapping configuration needed
- **Smart Routing**: Automatically detects LAN connections and uses direct private IP when possible

### Data Flow

1. Both client/server discover public and private IPs via STUN and local network detection
2. Post their network info to signaling server with shared room ID
3. Client retrieves server's network info from signaling server
4. Client determines best connection method (LAN vs WAN) based on network analysis
5. Client establishes port forwards to server using optimal connection path

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

- Each port mapping runs in its own goroutine (`handlePortMapping` function in run.go:53)
- Server mode uses continuous polling with presence refresh (run.go:147)
- Configuration parsing supports flexible string-to-struct conversion via custom UnmarshalJSON/UnmarshalYAML
- Network discovery combines STUN (public) and local interface detection (private)
- LAN optimization bypasses STUN when peers are detected on same network
- Graceful shutdown handling with context cancellation and signal handling