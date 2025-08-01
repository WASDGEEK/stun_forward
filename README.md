# STUN Forward

This tool creates a tunnel between two peers (a `sender` and a `receiver`) behind NATs, enabling functionality similar to port forwarding. It uses a signaling server to exchange peer connection information (discovered via STUN) to establish a direct P2P connection.

This is useful for exposing a service running on a machine behind a restrictive firewall or NAT to the public internet, without requiring manual port forwarding configuration on the router.

## Features

-   TCP and UDP forwarding
-   Configuration via a simple JSON file
-   NAT traversal using STUN
-   Independent PHP-based signaling server for easy deployment

## How It Works

1.  A **Signaling Server** (`index.php`) is deployed on a publicly accessible web server.
2.  Both the `sender` and `receiver` clients connect to a **STUN Server** (e.g., `stun.l.google.com:19302`) to discover their own public IP address and port.
3.  The clients post their public address and a shared **Room ID** to the signaling server.
4.  They then poll the signaling server to retrieve their peer's public address.
5.  Once the addresses are exchanged, they attempt to establish a direct P2P connection (hole punching).
6.  The `sender` listens on a local port and forwards all traffic to the `receiver`'s public address.
7.  The `receiver` forwards the traffic from the `sender` to a specified local service.

## Installation

### Client (`stun_forward`)

You need to have Go installed on your system.

Clone the repository and build the executable:

```bash
git clone https://github.com/WASDGEEK/stun_forward.git
cd stun_forward
go build -o stun_forward .
```

### Signaling Server (`index.php`)

You need a web server with PHP. Simply upload the `index.php` file to your server.

## Configuration

The `stun_forward` client is configured using a JSON file. Copy the `config.json.example` to `config.json` and edit it for your needs.

**`config.json` fields:**

-   `mode`: `"sender"` or `"receiver"`.
-   `room`: A secret string that both peers must share to be matched.
-   `signalURL`: The full URL to your `index.php` signaling server.
-   `stunServer`: (Optional) The STUN server to use. Defaults to `stun.l.google.com:19302`.
-   `mappings`: An array of port mapping strings in the format `"proto:localPort:remotePort"`.

## Usage

### 1. Create Configurations

Create two `config.json` files, one for the sender and one for the receiver.

**Receiver `config.json`:**
(Exposes local service on port 22, listens for peer on port 5000)
```json
{
  "mode": "receiver",
  "room": "mysecretssh",
  "signalURL": "http://your-server.com/index.php",
  "mappings": [
    "tcp:22:5000"
  ]
}
```

**Sender `config.json`:**
(Listens locally on port 5001, connects to peer on port 5000)
```json
{
  "mode": "sender",
  "room": "mysecretssh",
  "signalURL": "http://your-server.com/index.php",
  "mappings": [
    "tcp:5001:5000"
  ]
}
```

### 2. Run the Clients

On the receiver machine:
```bash
./stun_forward --config /path/to/receiver_config.json
```

On the sender machine:
```bash
./stun_forward --config /path/to/sender_config.json
```

### 3. Test the Connection

Now, any traffic sent to the `sender`'s local port `5001` will be forwarded to the `receiver`'s local service on port `22`.

```bash
ssh user@127.0.0.1 -p 5001
```
