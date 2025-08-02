// Package main - Signaling server communication
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SignalingClient handles communication with signaling server
type SignalingClient struct {
	client *http.Client
}

// NewSignalingClient creates a new signaling client
func NewSignalingClient() *SignalingClient {
	return &SignalingClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

// PostSignal sends signal data to signaling server
func (c *SignalingClient) PostSignal(url, role, room, data string) error {
	// Debug: Print what's being sent to signaling server
	log.Printf("DEBUG: PostSignal - URL: %s, Role: %s, Room: %s, DataLen: %d", url, role, room, len(data))
	
	body, err := json.Marshal(SignalingData{Role: role, Room: room, Data: data})
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-200 response (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// WaitForPeerData waits for peer data with exponential backoff
func (c *SignalingClient) WaitForPeerData(ctx context.Context, url, peerRole, room string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond
	maxBackoff := 5 * time.Second
	attempt := 0

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		attempt++
		resp, err := c.client.Get(fmt.Sprintf("%s?role=%s&room=%s", url, peerRole, room))
		if err != nil {
			// 网络错误，使用指数退避
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff = time.Duration(float64(backoff) * 1.5)
			}
			continue
		}

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				time.Sleep(backoff)
				continue
			}
			if len(body) > 0 {
				return string(body), nil
			}
		} else {
			resp.Body.Close()
		}

		// 成功请求但无数据，使用较短的等待时间
		waitTime := backoff
		if attempt <= 3 {
			waitTime = 200 * time.Millisecond // 前几次快速重试
		}
		
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return "", ctx.Err()
		}

		// 调整退避时间
		if backoff < maxBackoff {
			backoff = time.Duration(float64(backoff) * 1.2)
		}
	}
	return "", errors.New("timeout waiting for peer data")
}

// UpdateMappings sends updated mappings to signaling server
func (c *SignalingClient) UpdateMappings(url, room string, mappings []string) error {
	log.Printf("📤 Updating mappings to signaling server: %v", mappings)
	
	body, err := json.Marshal(map[string]interface{}{
		"room":     room,
		"mappings": mappings,
	})
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-200 response (%d): %s", resp.StatusCode, string(body))
	}
	
	log.Printf("✅ Mappings updated successfully")
	return nil
}

// CheckMappingUpdates checks for mapping updates from client (for server)
func (c *SignalingClient) CheckMappingUpdates(ctx context.Context, url, room string, lastMappingVersion int) (bool, string, error) {
	reqURL := fmt.Sprintf("%s?room=%s&role=client&check_updates=true&last_mapping_version=%d", 
		url, room, lastMappingVersion)
	
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return false, "", fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, "", fmt.Errorf("read response error: %w", err)
		}
		
		var updateInfo map[string]interface{}
		if err := json.Unmarshal(body, &updateInfo); err != nil {
			return false, "", fmt.Errorf("json unmarshal error: %w", err)
		}
		
		hasUpdate, _ := updateInfo["has_update"].(bool)
		clientData, _ := updateInfo["client_data"].(string)
		
		return hasUpdate, clientData, nil
	}
	
	return false, "", nil
}

// WatchMappingUpdates continuously watches for mapping updates
func (c *SignalingClient) WatchMappingUpdates(ctx context.Context, url, room string, callback func(string)) {
	lastMappingVersion := 0
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()
	
	log.Printf("👀 Starting mapping updates watcher for room: %s", room)
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("Mapping updates watcher stopped")
			return
		case <-ticker.C:
			hasUpdate, clientData, err := c.CheckMappingUpdates(ctx, url, room, lastMappingVersion)
			if err != nil {
				log.Printf("Error checking mapping updates: %v", err)
				continue
			}
			
			if hasUpdate && clientData != "" {
				log.Printf("🔄 Detected mapping updates from client")
				callback(clientData)
				lastMappingVersion = int(time.Now().Unix()) // Update to prevent re-processing
			}
		}
	}
}

// Close closes the signaling client
func (c *SignalingClient) Close() {
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}