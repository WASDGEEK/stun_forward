package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stun_forward/internal/config"
	"stun_forward/pkg/logger"
	"stun_forward/pkg/types"
)

const (
	AppName    = "stun_forward_v2"
	AppVersion = "2.0.0-dev"
)

func main() {
	// Parse command line arguments
	var (
		configPath = flag.String("config", "config.yml", "Path to configuration file")
		version    = flag.Bool("version", false, "Show version information")
		help       = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	// Show version and exit
	if *version {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		os.Exit(0)
	}

	// Show help and exit
	if *help {
		showHelp()
		os.Exit(0)
	}

	// Initialize logger
	log := logger.NewDefaultLogger().WithComponent("main")
	log.Info("Starting "+AppName, logger.String("version", AppVersion))

	// Initialize configuration manager
	configManager := config.NewManager()
	if err := configManager.LoadFromFile(*configPath); err != nil {
		log.Error("Failed to load configuration", logger.Error(err), logger.String("path", *configPath))
		os.Exit(1)
	}

	cfg := configManager.Get()
	log.Info("Configuration loaded successfully",
		logger.String("mode", string(cfg.Mode)),
		logger.String("roomId", cfg.RoomID),
		logger.Int("mappings", len(cfg.Mappings)))

	// Set log level from config
	log.SetLevel(logger.ParseLevel(cfg.LogLevel))

	// Create application context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize event bus
	eventBus := types.NewSimpleEventBus()
	defer eventBus.Close()

	// Subscribe to configuration changes
	configWatcher := configManager.Watch()
	go func() {
		for event := range configWatcher {
			log.Info("Configuration changed", logger.String("eventType", string(event.Type())))
			// Handle configuration changes here
		}
	}()

	// Start the application based on mode
	var app Application
	switch cfg.Mode {
	case types.ModeClient:
		app = NewClientApplication(cfg, log, eventBus)
	case types.ModeServer:
		app = NewServerApplication(cfg, log, eventBus)
	default:
		log.Error("Invalid mode", logger.String("mode", string(cfg.Mode)))
		os.Exit(1)
	}

	// Start the application
	if err := app.Start(ctx); err != nil {
		log.Error("Failed to start application", logger.Error(err))
		os.Exit(1)
	}

	log.Info("Application started successfully")

	// Wait for shutdown signal
	<-sigChan
	log.Info("Received shutdown signal, stopping...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop the application gracefully
	if err := app.Stop(shutdownCtx); err != nil {
		log.Error("Error during shutdown", logger.Error(err))
		os.Exit(1)
	}

	log.Info("Application stopped successfully")
}

// Application interface for both client and server applications
type Application interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ClientApplication implements the client-side application
type ClientApplication struct {
	config   *types.Config
	logger   logger.Logger
	eventBus types.EventBus
}

// NewClientApplication creates a new client application
func NewClientApplication(config *types.Config, log logger.Logger, eventBus types.EventBus) *ClientApplication {
	return &ClientApplication{
		config:   config,
		logger:   log.WithComponent("client"),
		eventBus: eventBus,
	}
}

// Start starts the client application
func (app *ClientApplication) Start(ctx context.Context) error {
	app.logger.Info("Starting client mode")
	
	// TODO: Implement client startup logic
	// 1. Network discovery
	// 2. Signaling connection
	// 3. Peer coordination
	// 4. Connection establishment
	// 5. Port forwarding setup
	
	app.logger.Info("Client started successfully",
		logger.Int("mappings", len(app.config.Mappings)))
	
	return nil
}

// Stop stops the client application
func (app *ClientApplication) Stop(ctx context.Context) error {
	app.logger.Info("Stopping client mode")
	
	// TODO: Implement client shutdown logic
	// 1. Stop port forwarding
	// 2. Close connections
	// 3. Disconnect from signaling
	
	app.logger.Info("Client stopped successfully")
	return nil
}

// ServerApplication implements the server-side application
type ServerApplication struct {
	config   *types.Config
	logger   logger.Logger
	eventBus types.EventBus
}

// NewServerApplication creates a new server application
func NewServerApplication(config *types.Config, log logger.Logger, eventBus types.EventBus) *ServerApplication {
	return &ServerApplication{
		config:   config,
		logger:   log.WithComponent("server"),
		eventBus: eventBus,
	}
}

// Start starts the server application
func (app *ServerApplication) Start(ctx context.Context) error {
	app.logger.Info("Starting server mode")
	
	// TODO: Implement server startup logic
	// 1. Network discovery
	// 2. Signaling connection
	// 3. Wait for clients
	// 4. Port allocation
	// 5. Connection establishment
	// 6. Service forwarding setup
	
	app.logger.Info("Server started successfully")
	return nil
}

// Stop stops the server application
func (app *ServerApplication) Stop(ctx context.Context) error {
	app.logger.Info("Stopping server mode")
	
	// TODO: Implement server shutdown logic
	// 1. Stop service forwarding
	// 2. Close connections
	// 3. Disconnect from signaling
	
	app.logger.Info("Server stopped successfully")
	return nil
}

// showHelp displays help information
func showHelp() {
	fmt.Printf(`%s v%s - Advanced P2P NAT Traversal Tool

USAGE:
    %s [OPTIONS]

OPTIONS:
    -config <path>    Path to configuration file (default: config.yml)
    -version          Show version information
    -help             Show this help message

EXAMPLES:
    # Start with default config
    %s

    # Start with custom config
    %s -config /path/to/my-config.yml

    # Show version
    %s -version

CONFIGURATION:
    The configuration file supports both YAML and JSON formats.
    
    Example client config:
        mode: client
        roomId: "my-room"
        signalingUrl: "https://signal.example.com/api"
        stunServer: "stun.l.google.com:19302"
        mappings:
          - "tcp:8080:80"
          - "udp:5353:53"
    
    Example server config:
        mode: server
        roomId: "my-room"
        signalingUrl: "https://signal.example.com/api"
        stunServer: "stun.l.google.com:19302"

For more information, visit: https://github.com/WASDGEEK/stun_forward
`, AppName, AppVersion, AppName, AppName, AppName, AppName)
}