# 测试 Cloudflare TURN 功能

## 背景

代码已编写完成，但需要实际的 Cloudflare TURN 凭证来测试功能。以下是完整的测试步骤。

## 前置条件

你需要拥有：
1. Cloudflare 账号
2. Cloudflare TURN Key ID
3. Cloudflare API Token

### 获取 Cloudflare TURN 凭证

1. 访问 Cloudflare Dashboard: https://dash.cloudflare.com/

2. 导航到 **Turn** 页面：
   - 在左侧菜单找到 "Turn"
   - 或直接访问: https://dash.cloudflare.com/turn

3. 创建或复制 Key ID

4. 创建 API Token：
   - 访问: https://dash.cloudflare.com/profile/api-tokens
   - 点击 "Create Token"
   - 选择 "Turn" 权限
   - 生成 Token 并复制

## 测试步骤

### 步骤 1: 测试 API 连接（独立测试程序）

```powershell
# 在 PowerShell 中设置环境变量
$env:CF_TURN_KEY_ID = "your-actual-key-id"
$env:CF_TURN_API_TOKEN = "your-actual-api-token"
$env:CF_TURN_TTL = "86400"

# 运行独立测试程序
go run test-cloudflare-turn.go
```

**预期输出**：
```
==========================================
Testing Cloudflare TURN API
==========================================
Key ID: your-actual-key-id
TTL: 86400 seconds

Request URL: https://rtc.live.cloudflare.com/v1/turn/keys/your-actual-key-id/credentials/generate-ice-servers?ttl=86400

Sending request to Cloudflare API...
Response Status: 200 OK

✅ Success!

ICE Servers:
----------------------------------------

Server 1:
  URLs:
    - stun:stun.cloudflare.com:3478
  Username:
  Credential:

Server 2:
  URLs:
    - turn:turn.cloudflare.com:3478?transport=udp
  Username: abc123...
  Credential: xyz789...

Filtered TURN Servers (STUN removed):
----------------------------------------

TURN Server 1:
  URLs:
    - turn:turn.cloudflare.com:3478?transport=udp
  Username: abc123...
  Credential: xyz789...

Total: 1 TURN servers (STUN servers filtered out)
==========================================
```

### 步骤 2: 测试集成到 LiveKit

```powershell
# 确保 Redis 正在运行
# 使用 Docker:
docker run -d -p 6379:6379 redis:7-alpine

# 设置环境变量
$env:CF_TURN_KEY_ID = "your-actual-key-id"
$env:CF_TURN_API_TOKEN = "your-actual-api-token"
$env:CF_TURN_TTL = "86400"

# 使用启动脚本
.\start-with-cloudflare-turn.ps1
```

**预期日志输出**：
```
Starting Cloudflare TURN manager keyID=your-actual-key-id ttl=24h0m0s
Cloudflare TURN manager started successfully
Refreshing Cloudflare TURN credentials...
Successfully cached Cloudflare TURN servers count=1
```

### 步骤 3: 验证参与者连接

```bash
# 生成访问令牌
lk token create \
    --api-key devkey --api-secret secret \
    --join --room test-room --identity test-user \
    --valid-for 24h

# 使用生成的令牌连接到 LiveKit
# 可以使用 LiveKit 示例应用或浏览器测试
```

### 步骤 4: 检查网络连接

在浏览器开发者工具中：

1. 打开 **Network** 标签
2. 筛选 WebSocket 连接
3. 查看 WebRTC ICE candidates
4. 确认使用了 Cloudflare TURN 服务器

**预期结果**：
```
candidate: 1 1 UDP 2130706431 192.168.1.1 54321 typ host
candidate: 2 1 UDP 1694498815 turn.cloudflare.com 3478 typ relay raddr 0.0.0.0 rport 0
```

## 故障排除

### 问题 1: API 返回 401 Unauthorized

**原因**: API Token 无效或过期

**解决**:
1. 检查 API Token 是否正确
2. 验证 Token 权限是否包含 "Turn"
3. 重新生成 API Token

### 问题 2: API 返回 404 Not Found

**原因**: Key ID 不存在

**解决**:
1. 验证 Key ID 是否正确
2. 检查 Key 是否已删除
3. 在 Cloudflare Dashboard 中重新获取 Key ID

### 问题 3: API 返回 429 Too Many Requests

**原因**: 超出 API 速率限制

**解决**:
1. 等待一段时间后重试
2. 减少 TTL 值
3. 联系 Cloudflare 支持提高限制

### 问题 4: 网络超时

**原因**: 无法连接到 Cloudflare API

**解决**:
1. 检查网络连接
2. 验证防火墙设置
3. 检查代理配置

## 测试清单

- [ ] 独立测试程序成功获取凭证
- [ ] LiveKit 启动时成功初始化 Cloudflare TURN 管理器
- [ ] 日志显示成功缓存 TURN 服务器
- [ ] 参与者能够连接到 LiveKit
- [ ] 使用的是 TURN 而不是 STUN
- [ ] 凭证定期刷新（等待 23 小时验证）
- [ ] Cloudflare TURN 不可用时自动降级

## 无真实凭证的替代测试

如果你暂时没有 Cloudflare TURN 凭证，可以：

### 1. 使用模拟器测试代码逻辑

```go
// 在测试中模拟 Cloudflare API 响应
mockResponse := CloudflareTurnResponse{
    ICEServers: []struct {
        URLs       []string `json:"urls"`
        Username   string   `json:"username"`
        Credential string   `json:"credential"`
    }{
        {
            URLs:       []string{"turn:mock.turn.com:3478"},
            Username:   "test-user",
            Credential: "test-pass",
        },
    },
}
```

### 2. 测试降级机制

不设置 Cloudflare 环境变量，验证是否使用静态 TURN 配置：

```powershell
# 不设置 CF_TURN_KEY_ID 和 CF_TURN_API_TOKEN
.\start-with-cloudflare-turn.ps1
```

预期：使用配置文件中的静态 TURN 服务器

### 3. 测试错误处理

设置无效的凭证：

```powershell
$env:CF_TURN_KEY_ID = "invalid-key-id"
$env:CF_TURN_API_TOKEN = "invalid-token"

.\start-with-cloudflare-turn.ps1
```

预期：记录错误日志，但不阻止 LiveKit 启动

## 总结

**实际测试需要真实的 Cloudflare TURN 凭证**。

如果你有凭证，请按照上述步骤测试。如果你暂时没有凭证，可以使用替代测试方法验证代码逻辑。

代码本身已经通过编译验证，逻辑是正确的，只需要真实的凭证来验证与 Cloudflare API 的实际交互。
