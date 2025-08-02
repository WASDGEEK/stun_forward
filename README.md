# STUN Forward

An advanced P2P port forwarding tool with **NAT hole punching** and **dynamic mapping management**. Create secure tunnels between clients and servers behind NATs without manual router configuration.

## ✨ Key Features

### 🚀 Advanced NAT Traversal
- **True UDP Hole Punching**: Real P2P connections using STUN
- **NAT Type Detection**: Full Cone, Restricted Cone, Port Restricted, Symmetric NAT
- **Multi-Strategy Connection**: Automatic fallback from P2P to relay
- **Smart LAN Detection**: Direct connection optimization for local networks

### 🎛️ Dynamic Configuration
- **Hot Mapping Updates**: Add/remove port mappings without restart
- **Interactive CLI**: Real-time mapping management interface
- **Auto Room Cleanup**: 5-minute inactivity cleanup for resource efficiency
- **Version Control**: Conflict-free concurrent updates

### 🔧 Protocol Support
- **UDP Hole Punching**: Direct P2P for supported NAT types
- **TCP Relay**: Reliable connection for all scenarios
- **Mixed Protocol**: Optimal connection method per mapping

### 📊 Enhanced Monitoring
- **Comprehensive Logging**: Detailed NAT detection and connection status
- **Real-time Updates**: Live mapping synchronization between client/server
- **Connection Analytics**: Performance metrics and hole punch success rates

## 🔄 How It Works

### Initial Connection
1. **Enhanced NAT Discovery**: Both client and server perform comprehensive NAT type detection
2. **Dynamic Port Allocation**: Server allocates available ports based on client requirements  
3. **Smart Connection Establishment**: Automatic selection of optimal connection method:
   - **LAN Direct**: Same network detection for optimal performance
   - **UDP Hole Punching**: P2P tunnels for compatible NAT types
   - **TCP Relay**: Fallback for complex NAT scenarios

### Real-time Management
4. **Live Mapping Updates**: Client can modify port mappings dynamically
5. **Instant Synchronization**: Server detects changes and reallocates ports automatically
6. **Seamless Reconnection**: New connections established without service interruption

## Quick Start

### 1. Install

```bash
git clone https://github.com/WASDGEEK/stun_forward.git
cd stun_forward
go build -o stun_forward .
```

### 2. Setup Signaling Server

Deploy the enhanced signaling server:
```bash
# Deploy signaling/signaling_server_enhanced.php to your web server
# Or use the basic version: signaling/signaling_server.php
```

The enhanced server provides:
- **Auto room cleanup** (5-minute inactivity timeout)
- **Real-time mapping synchronization** 
- **Version control** for conflict resolution

### 3. Configure

**Server (config.yml):**
```yaml
mode: server
roomId: "my-secret-room"
signalingUrl: "https://your-server.com/signaling_server_enhanced.php"
stunServer: "stun.l.google.com:19302"  # Optional, defaults to Google STUN
```

**Client (config.yml):**
```yaml
mode: client
roomId: "my-secret-room"  # Must match server
signalingUrl: "https://your-server.com/signaling_server_enhanced.php"
stunServer: "stun.l.google.com:19302"  # Optional
mappings:
  - "tcp:8080:22"    # Local 8080 -> Server 22 (SSH)
  - "udp:3306:3306"  # Local 3306 -> Server 3306 (MySQL, with hole punching)
  - "udp:5000:53"    # Local 5000 -> Server 53 (DNS, P2P optimized)
```

> 💡 **Example configs** available in `examples/configs/`

### 4. Run

**On the server machine:**
```bash
./stun_forward
```

**On the client machine:**
```bash
./stun_forward
```

Both will automatically use `config.yml` in the current directory.

### 5. Use & Manage

**Access server services through client:**
```bash
# SSH to server via client
ssh user@127.0.0.1 -p 8080

# Connect to MySQL on server via client (with UDP hole punching!)  
mysql -h 127.0.0.1 -P 3306 -u user -p
```

**Dynamic mapping management (client):**
```
mapping> help
Commands:
  add <protocol:localPort:remotePort> - Add new mapping
  remove <index> - Remove mapping by index  
  list - Show current mappings
  update - Send current mappings to server
  quit - Exit updater

mapping> add udp:6000:80
✅ Added mapping: udp 6000->80

mapping> update  
📤 Sending 4 mappings to server...
✅ Mapping update sent successfully
🎯 Server allocated new ports:
  udp 6000->80 allocated port: 45123
```

## Configuration Options

### Global Settings

- `mode`: `"client"` or `"server"`
- `roomId`: Shared secret for peer matching
- `signalingUrl`: URL to your signaling server (`index.php`)
- `stunServer`: STUN server for NAT traversal (optional, defaults to Google's)

### Client-Only Settings

- `mappings`: Array of port forwarding rules in format `"protocol:localPort:serverPort"`

### Supported Formats

Both YAML (`.yml`, `.yaml`) and JSON (`.json`) configuration files are supported.

## 🔧 Advanced Usage

### Custom Config File
```bash
./stun_forward --config /path/to/my-config.yml
```

### NAT Traversal Modes

The tool automatically selects the best connection method:

**🏠 LAN Direct** (Best Performance)
- Detected when client/server share same public IP
- Zero-latency local network communication
- Automatic fallback if detection fails

**🎯 UDP Hole Punching** (True P2P)
- Works with Full Cone and Restricted Cone NATs
- Direct peer-to-peer communication  
- Simultaneous connect with port prediction fallback

**🌐 TCP/UDP Relay** (Universal Compatibility)
- Guaranteed to work with any NAT type
- Fallback for Symmetric NAT or failed hole punching
- Reliable but higher latency

### Connection Analytics

Monitor your connections:
```
🔍 Network Discovery Results:
   Private: 192.168.1.100
   Public: 203.0.113.1:54321
   NAT Type: Full Cone NAT  
   Can Hole Punch: true
   Hole Punch Port: 45123

🎯 Using UDP hole punching for port 5000
✅ UDP hole punching established, proxying 5000 <-> P2P
```

## 🏗️ Architecture

### Enhanced P2P System

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │    │   Signaling  │    │   Server    │
│             │    │   Server     │    │             │
│ ┌─────────┐ │    │              │    │ ┌─────────┐ │
│ │ Mapping │ │◄──►│  Enhanced    │◄──►│ │ Dynamic │ │
│ │ Manager │ │    │  Protocol    │    │ │ Alloc   │ │
│ └─────────┘ │    │              │    │ └─────────┘ │
│             │    │ ┌──────────┐ │    │             │
│ ┌─────────┐ │    │ │ Auto     │ │    │ ┌─────────┐ │
│ │ NAT     │ │    │ │ Cleanup  │ │    │ │ Hole    │ │
│ │ Detect  │ │    │ │ 5min     │ │    │ │ Punch   │ │
│ └─────────┘ │    │ └──────────┘ │    │ └─────────┘ │
└─────────────┘    └──────────────┘    └─────────────┘
       │                                      │
       └──────── P2P Tunnel (UDP) ────────────┘
              or Relay (TCP/UDP)
```

### Data Flow Evolution

**🔄 Real-time Updates:**
```
Client Config Change → Signaling Server → Server Reallocation → New P2P/Relay
```

**🎯 Connection Establishment:**
```
NAT Detection → Connection Method Selection → P2P Hole Punch OR Relay Fallback
```

**📊 Multi-Strategy Approach:**
1. **LAN**: `Client:localPort ←→ Server:localPort` (direct)
2. **P2P**: `Client:localPort ←→ Server:allocatedPort` (hole punch)  
3. **Relay**: `Client:localPort → Relay → Server:allocatedPort`

## 🔒 Security & Production

### Security Best Practices
- **Strong Room IDs**: Use cryptographically secure random strings
- **HTTPS Signaling**: Always use TLS for signaling server communication
- **Network Isolation**: Consider VPN overlay for sensitive environments
- **Access Control**: Implement authentication at application layer

### Production Deployment
- **Enhanced Signaling Server**: Use `signaling_server_enhanced.php` for production
- **Load Balancing**: Multiple signaling servers with shared storage (Redis/Database)
- **Monitoring**: Track connection success rates and NAT traversal performance
- **Fallback Servers**: Multiple STUN servers for redundancy

## 🔧 Troubleshooting

### Connection Diagnostics

**Check NAT Type Detection:**
```
🔍 Network Discovery Results:
   NAT Type: Symmetric NAT  ❌ (Hole punching not possible)
   NAT Type: Full Cone NAT  ✅ (Optimal for hole punching)
```

**Monitor Connection Method:**
```
🏠 Using LAN connection (best performance)
🎯 Using UDP hole punching (P2P tunnel)  
🌐 Using TCP relay connection (fallback)
```

### Common Issues

**🚫 Hole Punching Fails**
- Check NAT type compatibility (avoid Symmetric NAT)
- Verify STUN server accessibility
- Try different STUN servers

**⏰ Connection Timeouts**
- Verify signaling server URL accessibility
- Check firewall rules for UDP/TCP ports
- Ensure room IDs match exactly

**🔄 Mapping Updates Not Syncing**
- Use enhanced signaling server (`signaling_server_enhanced.php`)
- Check client CLI commands are being sent (`update` command)
- Monitor server logs for mapping update detection

### Debug Logging

Enable verbose logging by checking output:
```
👀 Starting mapping updates watcher for room: my-room
🔄 Detected mapping updates from client  
✅ Successfully processed mapping update - 3 new port allocations
🎯 Using UDP hole punching for port 5000
```

### Performance Optimization

**Connection Priority:**
1. **LAN Direct** (0ms overhead)
2. **UDP Hole Punch** (minimal overhead)  
3. **TCP/UDP Relay** (higher latency)

**Mapping Strategies:**
- Use **UDP** for real-time applications (gaming, VoIP)
- Use **TCP** for reliable data transfer (file sharing, databases)
- Mix protocols based on application requirements

## 📁 Project Structure

```
stun_forward/
├── 📄 main.go                 # Entry point and configuration parsing
├── 📄 run.go                  # Core client/server logic and orchestration  
├── 📄 stun.go                 # Enhanced STUN discovery and NAT detection
├── 📄 holepunch.go            # UDP hole punching implementation
├── 📄 signaling.go            # Signaling server communication
├── 📄 forwarder.go            # Protocol-specific forwarding (TCP/UDP)
├── 📄 mapping_updater.go      # Dynamic mapping management
├── 📄 types.go                # Data structures and JSON/YAML parsing
├── 📁 signaling/
│   ├── signaling_server.php           # Basic signaling server
│   └── signaling_server_enhanced.php  # Enhanced server with auto-cleanup
└── 📁 examples/
    └── 📁 configs/
        ├── config.example.yml         # Example configuration
        ├── config.client.yml          # Client example
        ├── config.server.yml          # Server example  
        └── config.json.example        # JSON format example
```

## 🛠️ Technical Stack

- **Language**: Go 1.24+ with modern networking libraries
- **NAT Traversal**: STUN (RFC 5389) with custom hole punching
- **Signaling**: PHP-based REST API with enhanced protocol
- **Configuration**: YAML/JSON with flexible parsing
- **Networking**: UDP hole punching + TCP relay fallback
- **Concurrency**: Goroutine-based async I/O and connection management

## 📋 Requirements

- **Go**: 1.24+ for client/server binary
- **PHP**: 7.4+ with JSON support for signaling server  
- **Network**: Internet access for STUN discovery
- **Ports**: Dynamic allocation (no manual configuration needed)