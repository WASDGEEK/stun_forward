// signal_optimized.go - 优化版本的信令服务器通信
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

// OptimizedSignalClient 优化版的信令客户端
type OptimizedSignalClient struct {
	client *http.Client
}

// NewOptimizedSignalClient 创建优化的信令客户端
func NewOptimizedSignalClient() *OptimizedSignalClient {
	return &OptimizedSignalClient{
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

// PostSignalOptimized 优化版的信号发送
func (c *OptimizedSignalClient) PostSignalOptimized(url, role, room, data string) error {
	body, err := json.Marshal(SignalData{Role: role, Room: room, Data: data})
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

// WaitForPeerDataOptimized 优化版的peer数据等待，使用指数退避
func (c *OptimizedSignalClient) WaitForPeerDataOptimized(ctx context.Context, url, peerRole, room string, timeout time.Duration) (string, error) {
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

// Close 关闭信令客户端
func (c *OptimizedSignalClient) Close() {
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}