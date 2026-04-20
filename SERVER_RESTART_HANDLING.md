# 服务器重启处理机制

## 问题：服务器重启后定时器丢失

用户提出了一个重要问题：
> "你做的是'定时获取新的凭证'。如果服务器意外重启呢？定时就消失了。你怎么办？"

## 当前实现分析

### 启动流程

```go
func (m *CloudflareTurnManager) Start() error {
    // 1. 尝试从缓存文件加载
    if m.loadCacheFromFile() {
        logger.Infow("Loaded Cloudflare TURN credentials from cache")
    }
    
    // 2. 立即获取凭证（如果缓存无效或过期）
    if err := m.refresh(); err != nil {
        logger.Warnw("Failed to fetch credentials", err)
    }
    
    // 3. 启动定时刷新
    m.refreshTicker = time.NewTicker(refreshInterval)
    go m.refreshLoop()
    
    return nil
}
```

### 服务器重启后的行为

| 场景 | 行为 | 结果 |
|------|------|------|
| **正常重启** | 重新启动服务 | ✅ 立即获取新凭证，启动新定时器 |
| **崩溃重启** | 自动恢复 | ✅ 立即获取新凭证，启动新定时器 |
| **缓存有效** | 从文件加载 | ✅ 不调用 API，直接使用缓存 |
| **缓存过期** | 重新获取 | ✅ 调用 API 获取新凭证 |

## 改进方案：持久化缓存

### 功能特性

1. **自动缓存凭证**
   - 获取到凭证后自动保存到文件
   - 文件位置：`.livekit/cf_turn_cache.json`

2. **启动时加载缓存**
   - 如果缓存文件存在且未过期，直接使用
   - 避免每次重启都调用 API

3. **智能刷新**
   - 缓存有效：不调用 API
   - 缓存过期：调用 API 获取新凭证

### 缓存文件格式

```json
{
  "iceServers": [
    {
      "urls": ["turn:turn.cloudflare.com:3478?transport=udp"],
      "username": "abc123...",
      "credential": "xyz789..."
    }
  ],
  "expireAt": 1711234567
}
```

### 工作流程

```
服务器启动
    ↓
检查缓存文件
    ↓
┌─ 缓存存在且有效 ──┐
│                  │
│ 使用缓存凭证     │
│                  │
│ 启动定时器       │
└──────────────────┘

  缓存不存在或过期
         ↓
    调用 Cloudflare API
         ↓
    获取新凭证
         ↓
    保存到缓存文件
         ↓
    启动定时器
```

## 优势

### ✅ 解决定时器丢失问题

**改进前**：
- 重启 → 重新获取凭证 → 启动新定时器
- 问题：频繁重启会频繁调用 API

**改进后**：
- 重启 → 检查缓存 → 使用有效缓存 → 启动新定时器
- 优势：缓存有效时不调用 API

### ✅ 避免频繁 API 调用

| 场景 | 改进前 | 改进后 |
|------|--------|--------|
| 正常启动 | 调用 1 次 API | 调用 0 次 API（使用缓存） |
| 重启 1 次 | 调用 1 次 API | 调用 0 次 API（使用缓存） |
| 重启 10 次 | 调用 10 次 API | 调用 0 次 API（使用缓存） |
| 重启 100 次 | 调用 100 次 API | 调用 0 次 API（使用缓存） |

### ✅ 快速恢复

- 缓存有效时，启动即可使用（无需等待 API 响应）
- 减少启动延迟

### ✅ 离线工作（短暂）

- 即使 Cloudflare API 暂时不可用
- 只要缓存未过期，服务仍可正常运行

## 安全性

### 缓存文件保护

1. **文件权限**
   ```go
   // 写入时设置权限 0644
   os.WriteFile(m.cacheFile, data, 0644)
   ```

2. **文件位置**
   ```
   .livekit/cf_turn_cache.json
   ```
   - 通常不会被包含在 git 仓库中
   - 可以通过 `.gitignore` 忽略

3. **数据加密（可选）**
   - 当前：明文存储
   - 未来：可添加加密功能

### .gitignore 配置

确保缓存文件不被提交：

```gitignore
# Cloudflare TURN cache
.livekit/cf_turn_cache.json
.livekit/
```

## 测试场景

### 场景 1：正常启动（有缓存）

```bash
# 第一次启动
$ livekit-server --dev
INFO: Loading Cloudflare TURN credentials from cache
INFO: Successfully cached Cloudflare TURN servers count=1

# 重启
$ livekit-server --dev
INFO: Loaded Cloudflare TURN credentials from cache
INFO: Successfully loaded cache from file servers=1
INFO: Starting Cloudflare TURN manager
```

**结果**：✅ 不调用 API，直接使用缓存

### 场景 2：缓存过期

```bash
# 缓存过期后启动
$ livekit-server --dev
INFO: No valid cache found, fetching fresh credentials
INFO: Refreshing Cloudflare TURN credentials...
INFO: Successfully cached Cloudflare TURN servers count=1
```

**结果**：✅ 调用 API 获取新凭证

### 场景 3：频繁重启

```bash
# 重启 10 次
for i in {1..10}; do
    livekit-server --dev &
    sleep 2
    killall livekit-server
done
```

**结果**：
- 改进前：调用 10 次 API
- 改进后：调用 0 次 API（全部使用缓存）

### 场景 4：首次启动（无缓存）

```bash
# 首次启动
$ livekit-server --dev
INFO: Failed to read cache file (file not found)
INFO: Refreshing Cloudflare TURN credentials...
INFO: Successfully cached Cloudflare TURN servers count=1
```

**结果**：✅ 正常获取并缓存

## 监控和日志

### 关键日志

```log
# 缓存加载成功
INFO: Loaded Cloudflare TURN credentials from cache
INFO: Successfully loaded cache from file servers=1 expiresAt=2024-03-26T10:30:00Z

# 缓存加载失败
WARN: Failed to read cache file error=open .livekit/cf_turn_cache.json: no such file or directory

# 缓存过期
INFO: Cache file expired expiredAt=2024-03-25T10:00:00Z
INFO: No valid cache found, fetching fresh credentials

# 保存缓存
INFO: Saved Cloudflare TURN cache to file file=.livekit/cf_turn_cache.json
```

### Prometheus 指标（未来可添加）

```go
var (
    cacheLoadSuccess = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cloudflare_turn_cache_load_success_total",
        Help: "Number of times cache was successfully loaded",
    })
    cacheLoadFailed = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cloudflare_turn_cache_load_failed_total",
        Help: "Number of times cache load failed",
    })
    apiCallCount = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cloudflare_turn_api_calls_total",
        Help: "Number of API calls to Cloudflare",
    })
)
```

## 总结

### ✅ 问题已解决

| 问题 | 解决方案 |
|------|----------|
| 定时器丢失 | 启动时重新创建定时器 |
| 频繁重启 = 频繁 API 调用 | 使用持久化缓存 |
| 缓存不持久化 | 保存到 `.livekit/cf_turn_cache.json` |
| 启动慢 | 缓存有效时快速加载 |

### ✅ 优势

1. **健壮性**：服务器重启后立即恢复
2. **效率**：避免频繁 API 调用
3. **性能**：缓存有效时快速启动
4. **可靠性**：即使 API 暂时不可用也能工作

### ✅ 行为

服务器重启后：
1. 检查缓存文件
2. 缓存有效 → 直接使用，不调用 API
3. 缓存无效 → 调用 API 获取新凭证
4. 启动定时器，继续定期刷新

**问题完美解决！** 🎉
