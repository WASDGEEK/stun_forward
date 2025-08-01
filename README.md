# STUN Forward

This tool creates a tunnel between two peers (a `sender` and a `receiver`) behind NATs, enabling functionality similar to port forwarding. It uses a signaling server to exchange peer connection information to establish a direct connection.

This is useful for exposing a service running on a machine behind a restrictive firewall or NAT to the public internet, without requiring manual port forwarding configuration on the router.

## Features

-   TCP and UDP forwarding
-   Simple signaling server included
-   Cross-platform (macOS, Linux, Windows)

## How It Works

1.  Both the `sender` and `receiver` connect to a **Signaling Server** and identify themselves with a shared **Room ID**.
2.  They exchange their public IP addresses and port information through the signaling server.
3.  Once the connection information is exchanged, they attempt to establish a direct P2P connection.
4.  The `sender` listens on a local port for incoming connections/packets and forwards them to the `receiver`.
5.  The `receiver` receives the traffic from the `sender` and forwards it to a specified local service.

## Installation

You need to have Go installed on your system.

Clone the repository and build the executable:

```bash
git clone <your-repo-url>
cd stun_forward
go build -o stun_forward .
```

## Usage

The tool has three modes: `server`, `receiver`, and `sender`.

### 1. Run the Signaling Server

First, start the signaling server on a publicly accessible machine. This server acts as a rendezvous point for the peers.

```bash
./stun_forward -mode server -port 8080
```
The server will start listening on port 8080.

### 2. Run the Receiver

On the machine behind a NAT where the target service is running (e.g., an SSH server on port 22), start the `receiver`.

The `receiver` will connect to the signaling server and wait for the `sender`.

-   `tcp:22:5000`: Forward traffic from the peer (listening on its port 5000) to our local port 22.

```bash
./stun_forward \
    -mode receiver \
    -signal http://<your-signal-server-ip>:8080 \
    -room mysecretroom \
    -map tcp:22:5000
```

### 3. Run the Sender

On another machine (which can also be behind a NAT), start the `sender`. The `sender` will listen locally and forward traffic to the `receiver`.

-   `tcp:5001:5000`: Listen locally on port 5001 and forward all traffic to the peer's port 5000.

```bash
./stun_forward \
    -mode sender \
    -signal http://<your-signal-server-ip>:8080 \
    -room mysecretroom \
    -map tcp:5001:5000
```

### 4. Test the Connection

Now, any traffic sent to the `sender`'s local port `5001` will be forwarded to the `receiver`'s local service on port `22`.

For example, you can SSH into the `receiver`'s machine:

```bash
ssh user@127.0.0.1 -p 5001
```

## Example: `iperf3` Performance Test

#### On the Receiver machine:
1.  Start the `iperf3` server:
    ```bash
iperf3 -s -p 5201
    ```
2.  Start the `stun_forward` receiver to forward traffic to the `iperf3` server:
    ```bash
./stun_forward -mode receiver -room iperftest -signal http://<signal-ip>:8080 -map tcp:5201:5202
    ```

#### On the Sender machine:
1.  Start the `stun_forward` sender:
    ```bash
./stun_forward -mode sender -room iperftest -signal http://<signal-ip>:8080 -map tcp:5203:5202
    ```
2.  Run the `iperf3` client to connect to the sender, which forwards to the receiver's `iperf3` server:
    ```bash
iperf3 -c 127.0.0.1 -p 5203
```