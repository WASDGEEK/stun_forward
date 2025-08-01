// stun_optimized.go - 优化版本的STUN发现
package main

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
)

// STUNCache 缓存STUN发现结果
type STUNCache struct {
	publicAddr string
	timestamp  time.Time
	mutex      sync.RWMutex
}

var globalSTUNCache = &STUNCache{}

// getPublicIPOptimized 优化版的公网IP发现，支持结果缓存
func getPublicIPOptimized(stunServer string, cacheDuration time.Duration) (string, error) {
	// 先检查缓存
	globalSTUNCache.mutex.RLock()
	if time.Since(globalSTUNCache.timestamp) < cacheDuration && globalSTUNCache.publicAddr != "" {
		addr := globalSTUNCache.publicAddr
		globalSTUNCache.mutex.RUnlock()
		return addr, nil
	}
	globalSTUNCache.mutex.RUnlock()

	// 缓存过期或不存在，重新获取
	publicAddr, err := performSTUNDiscovery(stunServer)
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

// performSTUNDiscovery 执行实际的STUN发现（与原版相同的逻辑）
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

// ClearSTUNCache 清除STUN缓存（用于测试或强制刷新）
func ClearSTUNCache() {
	globalSTUNCache.mutex.Lock()
	globalSTUNCache.publicAddr = ""
	globalSTUNCache.timestamp = time.Time{}
	globalSTUNCache.mutex.Unlock()
}