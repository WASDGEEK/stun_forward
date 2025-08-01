// Package main - Signaling server communication
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// Close closes the signaling client
func (c *SignalingClient) Close() {
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}