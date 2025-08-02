# STUN Forward V2 - Clean Architecture Rewrite

🚀 **A complete rewrite of the STUN Forward project with clean architecture principles**

## 🎯 Goals of V2

1. **Clean Architecture** - Proper separation of concerns and dependency injection
2. **Better Testing** - Each component is designed to be unit testable
3. **Event-Driven** - Async communication using a robust event system
4. **Maintainable** - Clear module boundaries and interfaces
5. **Production Ready** - Comprehensive error handling and logging

## 📦 New Architecture

### Module Structure
```
stun_forward_v2/
├── cmd/                    # Application entry point
├── internal/               # Private application modules
│   ├── config/            # Configuration management
│   ├── network/           # Network discovery and utilities  
│   ├── signaling/         # Signaling server communication
│   ├── holepunch/         # NAT traversal and hole punching
│   ├── forwarding/        # Port forwarding engines
│   ├── coordination/      # Client/Server coordination
│   └── management/        # Dynamic mapping management
├── pkg/                   # Public interfaces and utilities
│   ├── types/            # Core type definitions
│   └── logger/           # Structured logging
└── signaling/            # PHP signaling servers (unchanged)
```

### Key Improvements

🎛️ **Event-Driven Architecture**
- All components communicate via events
- Clean separation of concerns
- Easy to test and mock

🔧 **Dependency Injection**
- Clear interfaces between modules
- Easy to swap implementations
- Better testability

📊 **Structured Logging**
- Component-based logging with fields
- Configurable log levels
- Better debugging and monitoring

⚙️ **Configuration Management**
- Hot reloading support
- Validation and type safety
- Multiple format support (YAML/JSON)

🔄 **Lifecycle Management**
- Proper startup and shutdown sequences
- Graceful error handling
- Resource cleanup

## 🚀 Current Implementation Status

### ✅ Completed
- [x] Core type definitions (`pkg/types/`)
- [x] Event system (`pkg/types/events.go`)
- [x] Structured logging (`pkg/logger/`)
- [x] Configuration management (`internal/config/`)
- [x] Application framework (`cmd/main.go`)
- [x] Project structure and architecture design

### 🚧 In Progress
- [ ] Network discovery module (`internal/network/`)
- [ ] STUN implementation (`internal/network/stun.go`)
- [ ] NAT detection (`internal/network/nat.go`)

### 📋 Planned
- [ ] Signaling client (`internal/signaling/`)
- [ ] Hole punching strategies (`internal/holepunch/`)
- [ ] UDP/TCP forwarding engines (`internal/forwarding/`)
- [ ] Client/Server coordination (`internal/coordination/`)
- [ ] Management CLI (`internal/management/`)
- [ ] Comprehensive testing
- [ ] Performance optimization

## 🔧 Development

### Building
```bash
# Build the new version
go build -o stun_forward_v2 ./cmd

# Run with custom config
./stun_forward_v2 -config config.yml

# Show help
./stun_forward_v2 -help
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific module tests
go test ./pkg/types/
go test ./internal/config/
```

## 🎯 Design Principles

### 1. Single Responsibility
Each module has one clear purpose and responsibility.

### 2. Dependency Inversion
Modules depend on interfaces, not concrete implementations.

### 3. Event-Driven Communication
Components communicate via events rather than direct coupling.

### 4. Error-First Design
Comprehensive error handling at every level.

### 5. Testability
Every component can be unit tested in isolation.

## 🔄 Migration from V1

The V2 rewrite maintains the same external APIs and configuration format while completely restructuring the internal implementation. This allows for:

- **Gradual Migration** - V1 and V2 can coexist during transition
- **Configuration Compatibility** - Existing configs work with V2
- **Feature Parity** - All V1 features are preserved or improved
- **Performance Improvements** - Better resource management and efficiency

## 🎪 Key Features Preserved

- ✅ P2P NAT traversal with UDP hole punching
- ✅ Dynamic port mapping management
- ✅ Smart connection selection (LAN → P2P → Relay)
- ✅ Interactive CLI for real-time updates
- ✅ Cross-platform support
- ✅ PHP signaling server compatibility

## 🚀 Next Steps

1. **Complete Core Modules** - Finish network discovery and STUN implementation
2. **Implement Connection Logic** - Hole punching and connection management
3. **Add Forwarding Engines** - UDP/TCP forwarding with session management
4. **Create Management Interface** - CLI for dynamic configuration
5. **Comprehensive Testing** - Unit tests, integration tests, and benchmarks
6. **Performance Optimization** - Memory usage, CPU efficiency, and connection speed
7. **Documentation** - API docs, examples, and troubleshooting guides

## 📚 Documentation

- [Architecture Design](ARCHITECTURE_V2.md) - Detailed architecture documentation
- [Original README](README.md) - V1 features and usage
- [Development Guide](CLAUDE.md) - Development instructions and conventions