// Package main - STUN discovery with caching support
package main

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
)

// stunCache caches STUN discovery results
type stunCache struct {
	publicAddr string
	timestamp  time.Time
	mutex      sync.RWMutex
}

var globalSTUNCache = &stunCache{}

// getPublicIP discovers public IP address with caching support, trying both IPv4 and IPv6
func getPublicIP(stunServer string, cacheDuration time.Duration) (string, error) {
	// 先检查缓存
	globalSTUNCache.mutex.RLock()
	if time.Since(globalSTUNCache.timestamp) < cacheDuration && globalSTUNCache.publicAddr != "" {
		addr := globalSTUNCache.publicAddr
		globalSTUNCache.mutex.RUnlock()
		return addr, nil
	}
	globalSTUNCache.mutex.RUnlock()

	// 缓存过期或不存在，重新获取 - 同时尝试IPv4和IPv6
	publicAddr, err := performDualStackSTUNDiscovery(stunServer)
	if err != nil {
		return "", err
	}

	// 更新缓存
	globalSTUNCache.mutex.Lock()
	globalSTUNCache.publicAddr = publicAddr
	globalSTUNCache.timestamp = time.Now()
	globalSTUNCache.mutex.Unlock()

	return publicAddr, nil
}

// performDualStackSTUNDiscovery tries both IPv4 and IPv6 STUN discovery
func performDualStackSTUNDiscovery(stunServer string) (string, error) {
	// Try IPv4 first (usually more reliable)
	if addr, err := performSTUNDiscoveryWithNetwork(stunServer, "udp4"); err == nil {
		return addr, nil
	}
	
	// If IPv4 fails, try IPv6
	if addr, err := performSTUNDiscoveryWithNetwork(stunServer, "udp6"); err == nil {
		return addr, nil
	}
	
	// If both fail, try original method (let system decide)
	return performSTUNDiscovery(stunServer)
}

// performSTUNDiscoveryWithNetwork performs STUN discovery with specific network type
func performSTUNDiscoveryWithNetwork(stunServer, network string) (string, error) {
	// Create a new UDP connection to the STUN server with specific network type
	conn, err := net.Dial(network, stunServer)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create a new STUN client
	client, err := stun.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a binding request
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	var publicAddr string
	// The callback function will be called when a response is received
	callback := func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			return
		}
		publicAddr = xorAddr.String()
	}

	// Send the request and wait for the response
	if err = client.Do(message, callback); err != nil {
		return "", err
	}

	if publicAddr == "" {
		return "", errors.New("failed to get public IP from STUN server")
	}

	return publicAddr, nil
}

// performSTUNDiscovery performs actual STUN discovery
func performSTUNDiscovery(stunServer string) (string, error) {
	// Create a new UDP connection to the STUN server.
	conn, err := net.Dial("udp", stunServer)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create a new STUN client.
	client, err := stun.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a binding request.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	var publicAddr string
	// The callback function will be called when a response is received.
	callback := func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			return
		}
		publicAddr = xorAddr.String()
	}

	// Send the request and wait for the response.
	if err = client.Do(message, callback); err != nil {
		return "", err
	}

	if publicAddr == "" {
		return "", errors.New("failed to get public IP from STUN server")
	}

	return publicAddr, nil
}

// clearSTUNCache clears STUN cache for testing or forced refresh
func clearSTUNCache() {
	globalSTUNCache.mutex.Lock()
	globalSTUNCache.publicAddr = ""
	globalSTUNCache.timestamp = time.Time{}
	globalSTUNCache.mutex.Unlock()
}