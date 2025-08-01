# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based P2P NAT traversal tool that creates tunnels between peers behind NATs using STUN for discovery and a PHP signaling server for coordination. It enables port forwarding without manual router configuration.

## Build and Development Commands

### Build
```bash
go build -o stun_forward .
```

### Run
```bash
./stun_forward --config /path/to/config.json
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

- **main.go**: Entry point, configuration validation, and CLI argument parsing
- **types.go**: Core data structures (`Config`, `PortMap`) with JSON unmarshaling
- **run.go**: Main execution logic, handles multiple port mappings concurrently
- **stun.go**: STUN client implementation for public IP discovery using github.com/pion/stun
- **signal.go**: HTTP client for signaling server communication (peer coordination)
- **tcp_udp.go**: Protocol-specific forwarding implementations

### Data Flow

1. Both sender/receiver discover public IPs via STUN server
2. Post their addresses to signaling server with shared room ID
3. Poll signaling server to get peer's address
4. Establish direct P2P connection (hole punching)
5. Forward traffic between local and remote ports

### Configuration

The tool uses JSON configuration files with these key fields:
- `mode`: "sender" or "receiver"
- `room`: Shared secret for peer matching
- `signalURL`: PHP signaling server endpoint
- `mappings`: Array of "proto:localPort:remotePort" strings

### Signaling Server

The `index.php` file implements a simple REST API for peer coordination:
- POST: Store peer address data
- GET: Retrieve peer address data by role and room

## Key Development Notes

- Each port mapping runs in its own goroutine (`handleMapping` function in run.go:26)
- Configuration parsing supports string-to-struct conversion for port mappings via custom UnmarshalJSON
- STUN discovery and signaling happen sequentially before establishing forwarding connections
- The tool blocks indefinitely using `select {}` after starting all goroutines