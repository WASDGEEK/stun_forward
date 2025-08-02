package types

import (
	"fmt"
	"net"
	"time"
)

// NATType represents different types of NAT
type NATType int

const (
	NATTypeUnknown NATType = iota
	NATTypeNone             // No NAT (direct internet connection)
	NATTypeFullCone         // Full Cone NAT (easiest to traverse)
	NATTypeRestrictedCone   // Restricted Cone NAT
	NATTypePortRestricted   // Port Restricted Cone NAT
	NATTypeSymmetric        // Symmetric NAT (hardest to traverse)
)

// String returns the string representation of NAT type
func (nt NATType) String() string {
	switch nt {
	case NATTypeNone:
		return "No NAT"
	case NATTypeFullCone:
		return "Full Cone NAT"
	case NATTypeRestrictedCone:
		return "Restricted Cone NAT"
	case NATTypePortRestricted:
		return "Port Restricted Cone NAT"
	case NATTypeSymmetric:
		return "Symmetric NAT"
	default:
		return "Unknown NAT"
	}
}

// CanHolePunch returns whether this NAT type supports hole punching
func (nt NATType) CanHolePunch() bool {
	switch nt {
	case NATTypeNone, NATTypeFullCone, NATTypeRestrictedCone, NATTypePortRestricted:
		return true
	case NATTypeSymmetric:
		return false // Generally difficult, but sometimes possible
	default:
		return false
	}
}

// ConnectionType represents the type of connection established
type ConnectionType int

const (
	ConnectionTypeLAN ConnectionType = iota
	ConnectionTypeP2P
	ConnectionTypeRelay
)

// String returns the string representation of connection type
func (ct ConnectionType) String() string {
	switch ct {
	case ConnectionTypeLAN:
		return "LAN"
	case ConnectionTypeP2P:
		return "P2P"
	case ConnectionTypeRelay:
		return "Relay"
	default:
		return "Unknown"
	}
}

// NetworkInfo contains comprehensive network information
type NetworkInfo struct {
	LocalIP    net.IP           `json:"localIp"`
	PublicIP   net.IP           `json:"publicIp"`
	PublicPort int              `json:"publicPort"`
	NATType    NATType          `json:"natType"`
	Endpoint   *net.UDPAddr     `json:"endpoint"`
	Timestamp  time.Time        `json:"timestamp"`
	
	// Additional discovery info
	STUNServer string           `json:"stunServer"`
	LocalPorts map[string]int   `json:"localPorts"` // Protocol -> port mappings
}

// String returns a human-readable representation
func (ni *NetworkInfo) String() string {
	return fmt.Sprintf("Local: %s, Public: %s:%d, NAT: %s, CanHolePunch: %v",
		ni.LocalIP, ni.PublicIP, ni.PublicPort, ni.NATType, ni.NATType.CanHolePunch())
}

// IsLAN checks if this network info represents a LAN connection to another network info
func (ni *NetworkInfo) IsLAN(other *NetworkInfo) bool {
	// Same public IP suggests they're behind the same NAT
	return ni.PublicIP.Equal(other.PublicIP)
}

// Target represents a connection target
type Target struct {
	Address        *net.UDPAddr
	NetworkInfo    *NetworkInfo
	ConnectionType ConnectionType
}

// ConnectionResult represents the result of a connection attempt
type ConnectionResult struct {
	Success        bool
	ConnectionType ConnectionType
	LocalAddr      *net.UDPAddr
	RemoteAddr     *net.UDPAddr
	Connection     net.Conn
	Error          error
	Duration       time.Duration
}

// String returns a human-readable representation
func (cr *ConnectionResult) String() string {
	if cr.Success {
		return fmt.Sprintf("Success: %s connection %s <-> %s (took %v)",
			cr.ConnectionType, cr.LocalAddr, cr.RemoteAddr, cr.Duration)
	}
	return fmt.Sprintf("Failed: %v (took %v)", cr.Error, cr.Duration)
}

// ForwardingStats represents statistics for port forwarding
type ForwardingStats struct {
	BytesIn       uint64        `json:"bytesIn"`
	BytesOut      uint64        `json:"bytesOut"`
	ConnectionsIn uint64        `json:"connectionsIn"`
	Errors        uint64        `json:"errors"`
	Uptime        time.Duration `json:"uptime"`
	LastActivity  time.Time     `json:"lastActivity"`
}

// HealthStatus represents the health status of a component
type HealthStatus int

const (
	HealthStatusHealthy HealthStatus = iota
	HealthStatusDegraded
	HealthStatusUnhealthy
)

// String returns the string representation of health status
func (hs HealthStatus) String() string {
	switch hs {
	case HealthStatusHealthy:
		return "Healthy"
	case HealthStatusDegraded:
		return "Degraded"
	case HealthStatusUnhealthy:
		return "Unhealthy"
	default:
		return "Unknown"
	}
}