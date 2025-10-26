# 企业微信 AI 机器人集成文档

## 概述

x-agent 支持与企业微信 AI 应用集成，用户可以通过企业微信与 Claude Code 进行对话交互。

## 功能特性

- ✅ 消息接收和转发
- ✅ 自动会话管理
- ✅ 命令支持 (/help, /new, /reset, /status)
- ✅ Markdown 格式支持
- ✅ 消息加密/解密
- ✅ Access Token 自动管理

## 配置步骤

### 1. 创建企业微信应用

1. 登录[企业微信管理后台](https://work.weixin.qq.com/)
2. 进入"应用管理" → "创建应用"
3. 填写应用信息并创建
4. 记录以下信息：
   - **CorpID**: 企业 ID（在"我的企业"页面查看）
   - **AgentID**: 应用 ID（在应用详情页查看）
   - **Secret**: 应用密钥（在应用详情页查看）

### 2. 配置接收消息服务器

1. 在应用详情页，找到"接收消息"配置
2. 设置以下参数：
   - **URL**: `https://your-domain.com/api/wechat/callback`
   - **Token**: 随机字符串（自己生成）
   - **EncodingAESKey**: 随机 43 位字符串（可点击"随机生成"）

### 3. 配置环境变量

创建 `.env` 文件或设置环境变量：

```bash
# 企业微信配置
export WECHAT_CORP_ID="ww1234567890abcdef"
export WECHAT_AGENT_ID="1000002"
export WECHAT_SECRET="your_secret_here"
export WECHAT_TOKEN="your_random_token"
export WECHAT_ENCODING_AES_KEY="your_43_char_key_here"
```

### 4. 启动服务

```bash
go run cmd/server/main.go
```

如果配置正确，你会看到日志：
```
WeChat Work integration enabled
```

### 5. 验证 URL

在企业微信管理后台保存接收消息配置时，系统会自动验证 URL 的有效性。

## 使用方法

### 基本对话

在企业微信中向应用发送消息，机器人会自动响应：

```
用户: 你好
机器人: 正在思考中...
机器人: 你好！我是 Claude Code AI 助手，有什么可以帮助你的吗？
```

### 命令列表

| 命令 | 说明 |
|------|------|
| `/help` | 显示帮助信息 |
| `/new` | 开始新对话（创建新会话） |
| `/reset` | 重置当前会话 |
| `/status` | 查看当前会话状态 |

### 使用示例

```
/help
可用命令:
/help - 显示帮助
/new - 开始新对话
/reset - 重置当前会话
/status - 查看会话状态
```

```
/status
会话 ID: a1b2c3d4
消息数: 5
创建时间: 2025-10-26 13:30:00
```

```
/new
已创建新会话: e5f6g7h8
```

## 技术架构

### 消息流程

```
企业微信用户
    ↓ (发送消息)
企业微信服务器
    ↓ (加密推送)
x-agent /api/wechat/callback
    ↓ (解密、验证)
WeChat Handler
    ↓ (处理命令 or 转发)
Session Manager + Container Manager
    ↓ (调用 Claude Code)
Claude Code Container
    ↓ (返回响应)
WeChat Handler
    ↓ (格式化、加密)
企业微信服务器
    ↓ (推送消息)
企业微信用户
```

### 核心组件

#### 1. Config (`internal/wechat/config.go`)
企业微信配置结构

#### 2. Crypto (`internal/wechat/crypto.go`)
消息加密/解密、签名验证

#### 3. Token Manager (`internal/wechat/token.go`)
Access Token 获取和缓存管理

#### 4. Handler (`internal/wechat/handler.go`)
消息接收、处理和发送

#### 5. User Session Store (`internal/wechat/session_store.go`)
企业微信用户 ID 与 Session ID 的映射管理

#### 6. Formatter (`internal/wechat/formatter.go`)
Markdown 格式转换

## 安全性

- ✅ 消息签名验证
- ✅ AES 加密/解密
- ✅ Access Token 安全存储
- ✅ CorpID 验证

## 故障排查

### 1. URL 验证失败

**问题**: 保存接收消息配置时提示 URL 验证失败

**解决方案**:
- 检查服务是否正常运行
- 检查域名是否可以从公网访问
- 检查 Token 和 EncodingAESKey 是否配置正确
- 查看服务日志中的错误信息

### 2. 无法收到消息

**问题**: 发送消息后机器人无响应

**解决方案**:
- 检查环境变量是否正确配置
- 检查应用是否已启用
- 查看服务日志确认是否收到消息
- 确认用户是否在应用的可见范围内

### 3. Access Token 错误

**问题**: 日志显示 Access Token 相关错误

**解决方案**:
- 检查 CorpID 和 Secret 是否正确
- 检查网络是否可以访问企业微信 API
- 查看具体的错误码和错误信息

## API 端点

### GET /api/wechat/callback

URL 验证端点（企业微信配置时使用）

**Query 参数**:
- `msg_signature`: 消息签名
- `timestamp`: 时间戳
- `nonce`: 随机数
- `echostr`: 加密的随机字符串

### POST /api/wechat/callback

消息接收端点

**Query 参数**:
- `msg_signature`: 消息签名
- `timestamp`: 时间戳
- `nonce`: 随机数

**Request Body**: 加密的 XML 消息

## 限制和注意事项

1. **消息长度**: 企业微信单条消息最大 2048 字节
2. **频率限制**: 注意企业微信 API 调用频率限制
3. **会话管理**: 每个企业微信用户对应一个会话
4. **容器资源**: 注意 Docker 容器资源使用情况

## 开发和测试

### 本地开发

使用 ngrok 等工具将本地服务暴露到公网：

```bash
ngrok http 8080
```

然后使用 ngrok 提供的 URL 配置企业微信回调地址。

### 单元测试

```bash
go test ./internal/wechat/...
```

## 参考资料

- [企业微信 API 文档](https://developer.work.weixin.qq.com/document/)
- [接收消息文档](https://developer.work.weixin.qq.com/document/path/90238)
- [消息加密解密文档](https://developer.work.weixin.qq.com/document/path/90968)
