# Cloudflare TURN 集成

## 概述

此功能允许 LiveKit 自动从 Cloudflare 获取短期 TURN 凭证，并每天自动刷新。这样避免了管理长期 TURN 凭证的安全风险。

## 功能特性

1. **自动凭证刷新**: 每天自动获取新的 Cloudflare TURN 凭证
2. **无需重启**: 新凭证自动生效，无需重启 LiveKit 服务
3. **智能过滤**: 自动过滤 STUN 服务器，只使用 TURN 服务器
4. **环境变量配置**: 通过环境变量轻松配置

## 环境变量

| 环境变量 | 必需 | 说明 | 默认值 |
|---------|------|------|--------|
| `CF_TURN_KEY_ID` | 是 | Cloudflare TURN Key ID | - |
| `CF_TURN_API_TOKEN` | 是 | Cloudflare API Token | - |
| `CF_TURN_TTL` | 否 | 凭证有效期（秒） | 86400 (24小时) |

## 使用方法

### 1. 配置环境变量

```bash
export CF_TURN_KEY_ID="your-cloudflare-turn-key-id"
export CF_TURN_API_TOKEN="your-cloudflare-api-token"
export CF_TURN_TTL="86400"  # 可选，默认 24 小时
```

### 2. 启动 LiveKit

```bash
livekit-server --config config.yaml
```

### 3. 验证

查看日志，应该能看到以下信息：

```
Starting Cloudflare TURN manager keyID=xxx ttl=24h0m0s
Cloudflare TURN manager started successfully
Refreshing Cloudflare TURN credentials...
Successfully cached Cloudflare TURN servers count=1
```

## 工作原理

### 启动时

1. 如果设置了 `CF_TURN_KEY_ID` 和 `CF_TURN_API_TOKEN`，Cloudflare TURN 管理器会自动启动
2. 首次启动时会立即获取 TURN 凭证
3. 凭证会被缓存，并在到期前自动刷新

### 运行时

1. 当新参与者加入时，LiveKit 会优先使用 Cloudflare TURN 凭证
2. 每 23 小时（默认）自动刷新凭证，确保持续可用
3. 新凭证自动替换旧凭证，无需重启服务

### 降级机制

如果 Cloudflare TURN 不可用（网络错误、API 错误等），LiveKit 会自动降级使用：
- 内置 TURN 服务器（如果启用）
- 配置文件中的静态 TURN 服务器（`rtc.turn_servers`）

## 注意事项

1. **网络要求**: LiveKit 服务器需要能够访问 `https://rtc.live.cloudflare.com`
2. **权限要求**: Cloudflare API Token 需要有足够的权限访问 TURN 服务
3. **时间同步**: 确保 LiveKit 服务器的时钟同步，以避免凭证过期问题
4. **静态配置**: 当使用 Cloudflare TURN 时，`rtc.turn_servers` 配置会被忽略

## 配置示例

### 完整配置文件 (config.yaml)

```yaml
port: 7880

rtc:
  port_range_start: 50000
  port_range_end: 60000
  tcp_port: 7881
  use_external_ip: true
  # 注意：当 CF_TURN_KEY_ID 设置时，turn_servers 配置会被忽略
  # turn_servers:
  #   - host: myhost.com
  #     port: 443
  #     protocol: tls
  #     username: ""
  #     credential: ""

keys:
  key1: secret1
```

### Docker 配置

```yaml
version: '3.8'
services:
  livekit:
    image: livekit/livekit-server:latest
    ports:
      - "7880:7880"
      - "7881:7881"
      - "50000-60000:50000-60000/udp"
    environment:
      - LIVEKIT_KEYS=key1:secret1
      - CF_TURN_KEY_ID=${CF_TURN_KEY_ID}
      - CF_TURN_API_TOKEN=${CF_TURN_API_TOKEN}
      - CF_TURN_TTL=${CF_TURN_TTL:-86400}
```

## 故障排除

### Cloudflare TURN 未启用

如果看到以下日志：
```
Failed to start Cloudflare TURN manager: xxx
```

检查：
1. 环境变量是否正确设置
2. 网络连接是否正常
3. Cloudflare API Token 是否有足够权限

### 凭证刷新失败

如果看到以下日志：
```
Failed to refresh Cloudflare TURN credentials: xxx
```

系统会自动重试，无需手动干预。如果问题持续，检查：
1. Cloudflare 服务状态
2. API Token 是否过期或被撤销
3. 网络连接是否稳定

## 安全建议

1. **保护 API Token**: 使用密钥管理服务（如 HashiCorp Vault、AWS Secrets Manager）存储 API Token
2. **最小权限原则**: 只授予 API Token 必要的权限
3. **定期轮换**: 定期更换 API Token
4. **监控告警**: 设置监控和告警，及时发现凭证获取失败的情况

## 性能影响

- **内存占用**: 极小，仅缓存凭证和少量元数据
- **网络开销**: 每 24 小时一次 HTTP 请求，约 1KB
- **CPU 占用**: 几乎没有影响，仅偶尔的 JSON 解析操作

## 参考资源

- [Cloudflare Turn Documentation](https://developers.cloudflare.com/turn/)
- [LiveKit Server Documentation](https://docs.livekit.io/)
- [WebRTC and TURN](https://webrtc.org/getting-started/turn-server)
