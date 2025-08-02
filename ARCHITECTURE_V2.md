# STUN Forward V2 - Clean Architecture Design

## 🎯 Core Principles

1. **Separation of Concerns** - Each module has a single responsibility
2. **Dependency Injection** - Clean interfaces between modules
3. **Event-Driven** - Async communication using channels
4. **Error-First** - Comprehensive error handling
5. **Testable** - Each component can be unit tested

## 📦 Module Structure

```
stun_forward_v2/
├── cmd/
│   └── main.go                 # Entry point and CLI
├── internal/
│   ├── config/                 # Configuration management
│   │   ├── config.go
│   │   └── validation.go
│   ├── network/               # Network discovery and utilities
│   │   ├── interface.go
│   │   ├── stun.go
│   │   └── nat.go
│   ├── signaling/             # Signaling server communication
│   │   ├── client.go
│   │   └── protocol.go
│   ├── holepunch/             # NAT traversal and hole punching
│   │   ├── manager.go
│   │   ├── strategies.go
│   │   └── connection.go
│   ├── forwarding/            # Port forwarding engines
│   │   ├── tcp.go
│   │   ├── udp.go
│   │   └── session.go
│   ├── coordination/          # Client/Server coordination
│   │   ├── client.go
│   │   ├── server.go
│   │   └── events.go
│   └── management/            # Dynamic mapping management
│       ├── cli.go
│       ├── updater.go
│       └── validator.go
├── pkg/                       # Public interfaces
│   ├── types/
│   │   ├── config.go
│   │   ├── network.go
│   │   └── events.go
│   └── logger/
│       └── logger.go
└── signaling/                 # Keep existing PHP servers
    ├── signaling_server.php
    └── signaling_server_enhanced.php
```

## 🔄 Data Flow Architecture

### 1. Event-Driven Core
```go
type EventBus interface {
    Publish(event Event)
    Subscribe(eventType EventType, handler EventHandler)
}

type Event interface {
    Type() EventType
    Data() interface{}
    Timestamp() time.Time
}
```

### 2. Connection Management
```go
type ConnectionManager interface {
    EstablishConnection(target Target) (Connection, error)
    GetConnection(id string) (Connection, bool)
    CloseConnection(id string) error
}

type Connection interface {
    ID() string
    Type() ConnectionType  // LAN, P2P, Relay
    Forward(data []byte) error
    Close() error
}
```

### 3. NAT Traversal Pipeline
```go
type HolePunchManager interface {
    DetectNAT() (*NATInfo, error)
    AttemptHolePunch(remote *NetworkInfo) (*P2PConnection, error)
    GetSupportedStrategies() []Strategy
}

type Strategy interface {
    Name() string
    CanAttempt(localNAT, remoteNAT *NATInfo) bool
    Execute(ctx context.Context, config *HolePunchConfig) (*ConnectionResult, error)
}
```

## 🎛️ Component Interfaces

### Configuration System
```go
type Config interface {
    Mode() Mode                    // Client or Server
    RoomID() string
    SignalingURL() string
    STUNServer() string
    Mappings() []PortMapping
    Validate() error
    Watch() <-chan ConfigEvent     // Hot reload
}
```

### Network Discovery
```go
type NetworkDiscovery interface {
    GetLocalIP() (net.IP, error)
    DetectNATType(stunServer string) (*NATInfo, error)
    GetPublicEndpoint(stunServer string) (*net.UDPAddr, error)
    IsLAN(localIP, remoteIP net.IP) bool
}
```

### Signaling Client
```go
type SignalingClient interface {
    Connect() error
    Register(info *NetworkInfo) error
    WaitForPeer() (*NetworkInfo, error)
    UpdateMappings(mappings []PortMapping) error
    Close() error
}
```

### Port Forwarding
```go
type Forwarder interface {
    Start(ctx context.Context, mapping PortMapping, target Target) error
    Stop() error
    Stats() *ForwardingStats
}
```

## 🚀 Execution Flow

### Client Mode
1. **Initialize** → Load config, setup logger, create event bus
2. **Network Discovery** → Detect local IP, NAT type, public endpoint
3. **Signaling** → Connect to server, register network info
4. **Wait for Server** → Receive server network info and port allocations
5. **Connection Establishment** → Try LAN → P2P → Relay
6. **Port Forwarding** → Start forwarders for each mapping
7. **Management Loop** → Handle config updates, health checks

### Server Mode
1. **Initialize** → Load config, setup logger, create event bus
2. **Network Discovery** → Detect local IP, NAT type, public endpoint
3. **Signaling** → Connect to server, register network info  
4. **Wait for Client** → Receive client network info and mapping requests
5. **Port Allocation** → Dynamically allocate ports for client mappings
6. **Connection Establishment** → Try LAN → P2P → Relay
7. **Service Forwarding** → Start forwarders to local services
8. **Management Loop** → Handle mapping updates, port reallocation

## 🔧 Key Improvements

### 1. Clean Error Handling
```go
type Error interface {
    error
    Code() ErrorCode
    Component() string
    Retryable() bool
}
```

### 2. Comprehensive Logging
```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithComponent(component string) Logger
}
```

### 3. Graceful Shutdown
```go
type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}
```

### 4. Metrics and Monitoring
```go
type Metrics interface {
    ConnectionCount() int
    ConnectionSuccess() float64
    TransferRate() float64
    NATType() string
}
```

## 🎯 Implementation Plan

1. **Phase 1**: Core infrastructure (config, logging, events)
2. **Phase 2**: Network discovery and STUN implementation
3. **Phase 3**: Signaling and coordination
4. **Phase 4**: Hole punching and connection management
5. **Phase 5**: Port forwarding engines
6. **Phase 6**: Management CLI and hot updates
7. **Phase 7**: Testing and optimization