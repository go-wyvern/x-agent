# 企业微信智能机器人集成文档

## 概述

x-agent 支持与企业微信智能机器人集成，用户可以在群聊中通过 @mention 机器人与 Claude Code 进行对话交互。

## 功能特性

- ✅ 智能机器人 API 模式（群聊支持）
- ✅ @mention 被动回复
- ✅ 群聊消息接收和转发
- ✅ 自动会话管理（支持群聊共享会话）
- ✅ 命令支持 (/help, /new, /reset, /status)
- ✅ Markdown 格式支持
- ✅ Webhook 消息推送

## 配置步骤

### 1. 创建企业微信智能机器人

1. 登录[企业微信管理后台](https://work.weixin.qq.com/)
2. 进入"应用管理" → "智能机器人"
3. 点击"创建机器人"
4. 填写机器人信息：
   - **机器人名称**: x-agent（或自定义名称）
   - **描述**: Claude Code AI 助手
   - **头像**: 上传机器人头像

### 2. 配置 Webhook 回调

1. 在机器人详情页，找到"接收消息"配置
2. 设置 Webhook 回调地址：
   - **URL**: `https://your-domain.com/api/wechat/webhook`
3. 记录 **Webhook Key**（用于消息验证）

### 3. 配置环境变量

创建 `.env` 文件或设置环境变量：

```bash
# 企业微信智能机器人配置
export WECHAT_WEBHOOK_KEY="your_webhook_key_here"
```

### 4. 启动服务

```bash
go run cmd/server/main.go
```

如果配置正确，你会看到日志：
```
WeChat Work Smart Robot integration enabled
```

### 5. 将机器人添加到群聊

1. 在企业微信中打开目标群聊
2. 点击群设置 → 群机器人
3. 选择并添加你创建的智能机器人
4. 机器人成功加入群聊后即可使用

## 使用方法

### 群聊中使用

在群聊中通过 @mention 机器人来对话：

```
用户: @x-agent 你好
机器人: 正在思考中...
机器人: 你好！我是 Claude Code AI 助手，有什么可以帮助你的吗？
```

**注意**: 机器人只会响应包含 @mention 的消息，不会干扰群聊的正常对话。

### 命令列表

| 命令 | 说明 |
|------|------|
| `@x-agent /help` | 显示帮助信息 |
| `@x-agent /new` | 开始新对话（创建新会话） |
| `@x-agent /reset` | 重置当前会话 |
| `@x-agent /status` | 查看当前会话状态 |

### 使用示例

#### 查看帮助
```
用户: @x-agent /help
机器人: 可用命令:
/help - 显示帮助
/new - 开始新对话
/reset - 重置当前会话
/status - 查看会话状态
```

#### 查看会话状态
```
用户: @x-agent /status
机器人: 会话 ID: a1b2c3d4
消息数: 5
创建时间: 2025-10-26 13:30:00
```

#### 创建新会话
```
用户: @x-agent /new
机器人: 已创建新会话: e5f6g7h8
```

#### 编程问答
```
用户: @x-agent 写一个 Go 语言的 Hello World 程序
机器人: 正在思考中...
机器人: [提供代码示例和解释]
```

## 技术架构

### 消息流程

```
企业微信用户（群聊中 @mention）
    ↓
企业微信服务器
    ↓ (Webhook 推送)
x-agent /api/wechat/webhook
    ↓ (JSON 解析)
WeChat Handler
    ↓ (@mention 检测 + 处理命令/转发)
Session Manager + Container Manager
    ↓ (调用 Claude Code)
Claude Code Container
    ↓ (返回响应)
WeChat Handler
    ↓ (格式化消息)
企业微信服务器 (Webhook URL)
    ↓ (推送消息)
企业微信群聊
```

### 核心组件

#### 1. Config (`internal/wechat/config.go`)
智能机器人配置结构（Webhook Key）

#### 2. Crypto (`internal/wechat/crypto.go`)
Webhook 消息验证（简化的 SHA256 签名）

#### 3. Handler (`internal/wechat/handler.go`)
- Webhook 消息接收和处理
- @mention 检测和内容提取
- 群聊/单聊消息路由
- 命令处理
- 消息发送（通过 Webhook URL）

#### 4. User Session Store (`internal/wechat/session_store.go`)
- 用户/群聊 ID 与 Session ID 的映射管理
- 群聊共享会话支持

#### 5. Formatter (`internal/wechat/formatter.go`)
Markdown 格式转换

#### 6. Types (`internal/wechat/types.go`)
- `WebhookMessage`: 接收的消息结构
- `WebhookResponse`: 回复的消息结构

## 会话管理策略

### 群聊模式

**共享会话（当前实现）**:
- 一个群聊对应一个共享会话（使用 `ChatID` 作为 session key）
- 所有群成员的对话共享同一个上下文
- 优点: 上下文连贯，适合协作讨论
- 注意: 多人同时对话可能导致上下文混乱

**会话隔离**:
如需每个用户在群聊中有独立会话，可修改 `handler.go` 中的 `sessionKey` 逻辑：
```go
// 当前: sessionKey = chatID
// 改为: sessionKey = fmt.Sprintf("%s_%s", chatID, userID)
```

### 单聊模式

智能机器人也支持一对一对话（如果企业微信支持）：
- 使用 `UserID` 作为 session key
- 每个用户有独立的会话和容器

## 安全性

- ✅ Webhook Key 验证
- ✅ 消息签名校验（SHA256）
- ✅ @mention 过滤（只处理 @mention 的消息）
- ✅ 异步消息处理

## 故障排查

### 1. Webhook 配置失败

**问题**: 保存 Webhook 配置时提示验证失败

**解决方案**:
- 检查服务是否正常运行（`:8080`）
- 检查域名是否可以从公网访问
- 检查 `WECHAT_WEBHOOK_KEY` 是否配置正确
- 查看服务日志中的错误信息

### 2. 机器人不响应

**问题**: @mention 机器人后无响应

**解决方案**:
- 检查是否正确 @mention 了机器人（`@x-agent`）
- 确认机器人已成功添加到群聊
- 查看服务日志确认是否收到 Webhook 消息
- 检查 Webhook URL 是否正确配置

### 3. 消息格式错误

**问题**: 日志显示 JSON 解析错误

**解决方案**:
- 检查企业微信发送的消息格式
- 查看 `types.go` 中的消息结构定义是否匹配
- 启用详细日志查看原始消息内容

## API 端点

### POST /api/wechat/webhook

Webhook 消息接收端点（智能机器人模式）

**Request Body**: JSON 格式的 Webhook 消息
```json
{
  "WebhookUrl": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
  "ChatId": "wrAAAAAAABgQAAAAAAAAAA",
  "ChatType": "group",
  "From": {
    "UserId": "zhangsan",
    "Name": "张三",
    "Alias": "zhangsan"
  },
  "MsgId": "123456789",
  "MsgType": "text",
  "Text": {
    "Content": "@x-agent 你好"
  }
}
```

**Response**: JSON 格式
```json
{
  "code": 0
}
```

## 限制和注意事项

1. **@mention 必须**: 群聊中机器人只响应包含 @mention 的消息
2. **消息长度**: 企业微信单条消息最大限制
3. **会话管理**: 群聊默认共享会话，可能出现上下文混乱
4. **容器资源**: 注意 Docker 容器资源使用情况
5. **异步处理**: 消息处理是异步的，避免 Webhook 超时

## 开发和测试

### 本地开发

使用 ngrok 等工具将本地服务暴露到公网：

```bash
ngrok http 8080
```

然后使用 ngrok 提供的 URL 配置企业微信 Webhook 地址。

### 单元测试

```bash
go test ./internal/wechat/...
```

## 与旧版本的区别

### 旧版本（Custom Application 模式）
- 复杂的 AES-256-CBC 加密/解密
- Access Token 管理
- 需要配置 CorpID, AgentID, Secret, Token, EncodingAESKey
- 不支持群聊（需要 App Chat API）
- 用户需要在"工作台"中找到应用

### 新版本（Smart Robot 模式）
- ✅ 简化的 Webhook Key 验证
- ✅ 无需 Access Token 管理
- ✅ 只需配置 Webhook Key
- ✅ 原生群聊支持（@mention）
- ✅ 直接添加到任意群聊

## 参考资料

- [企业微信 API 文档](https://developer.work.weixin.qq.com/document/)
- [智能机器人 API 文档](https://developer.work.weixin.qq.com/document/path/101039)
- [群机器人配置说明](https://developer.work.weixin.qq.com/document/path/90235)
