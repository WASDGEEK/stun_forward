# STUN Forward

A simple P2P port forwarding tool that creates secure tunnels between a **client** and **server** behind NATs, enabling direct access to services without manual router configuration.

## Features

- **Simple Client/Server Model**: Client decides port mappings, server just runs
- **Smart Connection**: Automatically detects LAN and uses direct connection when possible
- **TCP and UDP Support**: Forward any protocol
- **YAML Configuration**: Human-friendly config files
- **NAT Traversal**: Uses STUN for public internet connections
- **Zero Configuration Server**: Server needs no port mapping setup

## How It Works

1. **Server** starts and registers its network info (both public and private IPs) with the signaling server
2. **Client** connects and discovers the server's network information
3. **Smart Routing**: If both are on the same LAN, uses direct local connection; otherwise uses STUN for NAT traversal
4. **Port Forwarding**: Client listens on specified local ports and forwards traffic to server ports

## Quick Start

### 1. Install

```bash
git clone https://github.com/WASDGEEK/stun_forward.git
cd stun_forward
go build -o stun_forward .
```

### 2. Setup Signaling Server

Upload `index.php` to any web server with PHP support.

### 3. Configure

**Server (config.yml):**
```yaml
mode: server
roomId: "my-secret-room"
signalingUrl: "http://your-server.com/signal.php"
```

**Client (config.yml):**
```yaml
mode: client
roomId: "my-secret-room"  # Must match server
signalingUrl: "http://your-server.com/signal.php"
mappings:
  - "tcp:8080:22"    # Local 8080 -> Server 22 (SSH)
  - "tcp:3306:3306"  # Local 3306 -> Server 3306 (MySQL)
  - "udp:5000:53"    # Local 5000 -> Server 53 (DNS)
```

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

### 5. Use

Now you can access server services through client:
```bash
# SSH to server via client
ssh user@127.0.0.1 -p 8080

# Connect to MySQL on server via client  
mysql -h 127.0.0.1 -P 3306 -u user -p
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

## Advanced Usage

### Custom Config File

```bash
./stun_forward --config /path/to/my-config.yml
```

### LAN Optimization

The tool automatically detects when client and server are on the same LAN and uses direct connection, bypassing STUN for better performance and reliability.

### Multiple Services

Add as many port mappings as needed in the client config:

```yaml
mappings:
  - "tcp:2222:22"    # SSH
  - "tcp:8080:80"    # HTTP
  - "tcp:8443:443"   # HTTPS
  - "udp:5353:53"    # DNS
  - "tcp:5432:5432"  # PostgreSQL
```

## Architecture

### Dynamic Port Allocation

The tool uses an intelligent dynamic port allocation system to avoid port conflicts:

- **Client**: Defines port mappings and listens on local ports
- **Server**: Dynamically allocates available ports for each mapping, forwards to local services
- **Signaling Server**: Coordinates port allocation information between client and server
- **STUN Server**: External service for NAT traversal (only used when needed)

### Data Flow

```
Local App → Client:localPort → P2P → Server:allocatedPort → Local Service:targetPort
```

**Example with mapping `"tcp:11145:5201"`:**
```
iperf3 client → localhost:11145 → P2P → server:45678 → iperf3 server:5201
```

The server automatically allocates port 45678 (or any available port) to avoid conflicts with the iperf3 server already running on port 5201.

## Security Notes

- Use strong, unique `roomId` values
- Run signaling server over HTTPS in production  
- Consider VPN for sensitive data
- The tool creates direct P2P connections when possible

## Troubleshooting

### Connection Issues
- Ensure both client and server use the same `roomId`
- Check that signaling server is accessible from both sides
- Verify firewall settings allow the application

### LAN Detection
- Tool automatically prefers LAN connections when available
- Check logs to see whether LAN or WAN connection is being used
- Private IP detection works for standard ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x)

### Port Conflicts
- Ensure local ports on client are not already in use
- Server automatically allocates available ports to avoid conflicts
- Check server logs to see which ports were allocated for each mapping

### Server Startup Process
1. Server starts and registers with signaling server
2. Server waits for client to send mapping configuration
3. Server dynamically allocates ports for each client mapping
4. Server starts listeners on allocated ports
5. Server sends port allocation info back to client
6. Client connects to allocated ports for forwarding

### Example Server Log Output
```
[server] Received client registration with 3 mappings
[server] Allocated tcp port 45678 for client mapping 11145->5201
[server] Allocated tcp port 45679 for client mapping 8080->22
[server] Starting tcp server on allocated port 45678 -> local service 127.0.0.1:5201
[server] Server ready! All 3 port listeners started.
```