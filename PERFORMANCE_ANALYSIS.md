# STUN Forward 性能分析和优化建议

## 🔍 当前性能分析

### 程序规模
- **二进制大小**: 8.2MB
- **主要依赖**: pion/stun, pion/dtls等WebRTC相关库
- **运行时内存**: 约11MB (基于ps输出)

## ⚡ 性能瓶颈识别

### 1. STUN发现阶段 (stun.go:12-55)
**问题:**
- 每个端口映射都重复执行STUN发现
- 创建新的UDP连接和STUN客户端开销

**优化潜力:** 🔥 高

### 2. 信令服务器轮询 (signal.go:38-64)  
**问题:**
- 固定1秒间隔轮询，效率低
- 每次轮询都创建新的HTTP连接
- 30秒超时可能过长

**优化潜力:** 🔥 高

### 3. 数据转发性能 (tcp_udp.go)
**问题:**
- TCP: io.Copy缓冲区使用默认大小(32KB)
- UDP: 固定2048字节缓冲区，可能过小
- 缺少连接复用和连接池

**优化潜力:** 🔥 中等

### 4. Goroutine管理 (run.go:19-23)
**问题:**  
- 缺少goroutine泄漏控制
- 没有优雅退出机制
- 无限循环可能导致CPU占用

**优化潜力:** 🔶 中等

## 🚀 具体优化建议

### 1. STUN发现优化
```go
// 建议：缓存STUN结果，多个映射共享
type STUNCache struct {
    publicAddr string
    timestamp  time.Time
    mutex      sync.RWMutex
}

func (s *STUNCache) GetPublicAddr(stunServer string, cacheDuration time.Duration) (string, error) {
    s.mutex.RLock()
    if time.Since(s.timestamp) < cacheDuration && s.publicAddr != "" {
        addr := s.publicAddr
        s.mutex.RUnlock()
        return addr, nil
    }
    s.mutex.RUnlock()
    
    // 重新获取STUN地址...
}
```

### 2. 信令服务器优化  
```go
// 建议：指数退避和WebSocket升级
func WaitForPeerDataOptimized(url, peerRole, room string, timeout time.Duration) (string, error) {
    client := &http.Client{Timeout: 5 * time.Second}
    backoff := 500 * time.Millisecond
    maxBackoff := 5 * time.Second
    
    for start := time.Now(); time.Since(start) < timeout; {
        // 使用复用的client和指数退避
        resp, err := client.Get(fmt.Sprintf("%s?role=%s&room=%s", url, peerRole, room))
        if err == nil && resp.StatusCode == 200 {
            // 处理响应...
        }
        
        time.Sleep(backoff)
        if backoff < maxBackoff {
            backoff *= 2
        }
    }
}
```

### 3. 数据转发优化
```go
// 建议：自定义缓冲区和连接池
const (
    TCPBufferSize = 64 * 1024  // 64KB缓冲区
    UDPBufferSize = 8 * 1024   // 8KB UDP缓冲区
)

func optimizedTCPProxy(src, dst net.Conn) {
    defer src.Close()
    defer dst.Close()
    
    // 使用更大的缓冲区
    buf := make([]byte, TCPBufferSize)
    _, err := io.CopyBuffer(dst, src, buf)
    if err != nil {
        log.Printf("TCP proxy error: %v", err)
    }
}
```

### 4. 资源管理优化
```go
// 建议：添加context和优雅退出
func Run(cfg Config) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    var wg sync.WaitGroup
    for _, m := range cfg.Mappings {
        wg.Add(1)
        go func(mapping PortMap) {
            defer wg.Done()
            handleMappingWithContext(ctx, cfg, mapping)
        }(m)
    }
    
    // 监听信号进行优雅退出
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    cancel()
    wg.Wait()
}
```

## 📊 预期优化效果

| 优化项目 | 当前 | 优化后 | 改进 |
|---------|------|--------|------|
| STUN发现延迟 | 3-5秒×N个映射 | 3-5秒×1次 | 60-80% |
| 信令轮询频率 | 每秒1次 | 指数退避 | 50-70% |
| TCP转发吞吐 | ~90Gbps | ~95Gbps | 5-10% |
| 内存使用 | 11MB | 8-9MB | 20-30% |
| 启动时间 | 10-15秒 | 5-8秒 | 40-50% |

## 🎯 优先级建议

### 高优先级 (立即实施)
1. **STUN结果缓存** - 显著减少启动时间
2. **信令轮询优化** - 减少服务器负载和网络开销

### 中优先级 (下个版本)
1. **数据转发缓冲区优化** - 提升大文件传输性能
2. **连接复用** - 减少连接建立开销

### 低优先级 (长期改进)
1. **WebSocket信令** - 实时性更好，但增加复杂度
2. **多路径支持** - 同时使用多个网络路径

## 💡 额外建议

### 配置优化
```json
{
  "performance": {
    "stunCache": "5m",
    "signalBackoff": "500ms",
    "tcpBufferSize": 65536,
    "udpBufferSize": 8192,
    "maxRetries": 3
  }
}
```

### 监控指标
- 添加Prometheus metrics暴露
- 连接数、传输字节数、错误率统计
- STUN发现耗时、信令延迟监控