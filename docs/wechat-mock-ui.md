# 企业微信智能机器人 Mock UI 使用文档

## 概述

这是一个用于本地测试企业微信智能机器人 Webhook 回调的 Mock 前端界面，使用 Vue 3 + TypeScript 开发（单文件 HTML 实现）。

## 功能特性

- ✅ **双聊天界面**: 支持私聊和群聊两种模式
- ✅ **双模式支持**: 支持 Webhook (旧版) 和 Callback (加密) 两种模式
- ✅ **消息加密**: 支持 AES-256-CBC 加密/解密和 SHA1 签名验证
- ✅ **URL 验证**: 支持企业微信回调 URL 验证测试
- ✅ **实时响应显示**: 在聊天界面中显示服务器响应内容
- ✅ **配置灵活**: 可自定义 Webhook/Callback 地址、Token、EncodingAESKey 等
- ✅ **快捷命令**: 支持 `/help`、`/new`、`/status`、`/reset` 等命令
- ✅ **@mention 支持**: 群聊模式自动处理 @x-agent 提及
- ✅ **配置持久化**: 使用 LocalStorage 保存配置信息

## 快速开始

### 1. 配置环境变量

在项目根目录创建 `.env` 文件并配置以下环境变量：

```bash
# Callback 模式（加密）所需
WECHAT_TOKEN=your_token_here
WECHAT_ENCODING_AES_KEY=abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG

# 可选配置
SERVER_PORT=8080
```

**注意**: `WECHAT_ENCODING_AES_KEY` 必须是 43 位字符（Base64 编码后为 32 字节）

### 2. 启动 x-agent 服务

确保 x-agent 服务正在运行：

```bash
cd /path/to/x-agent
go run cmd/server/main.go
```

服务默认监听在 `http://localhost:8080`

### 3. 打开 Mock UI

在浏览器中直接打开 `wechat-mock-ui.html` 文件：

```bash
# 方法 1: 直接用浏览器打开
open wechat-mock-ui.html  # macOS
xdg-open wechat-mock-ui.html  # Linux
start wechat-mock-ui.html  # Windows

# 方法 2: 使用简单的 HTTP 服务器
python3 -m http.server 8000
# 然后访问 http://localhost:8000/wechat-mock-ui.html

# 方法 3: 使用 npx
npx serve .
```

## 界面说明

### 左侧面板 - 私聊模式

模拟与机器人的一对一私聊：

- **发送消息**: 直接输入内容发送，无需 @mention
- **快捷命令**: 点击快捷按钮快速输入常用命令
- **消息显示**: 显示用户消息、机器人回复和系统提示

### 中间面板 - 群聊模式

模拟群聊场景：

- **@mention 必须**: 消息需要包含 `@x-agent` 才会被机器人处理
- **快捷命令**: 快捷按钮会自动添加 `@x-agent` 前缀
- **群聊特性**: 模拟多人群聊环境

### 右侧面板 - 配置

#### 模式选择

- **Webhook (旧版)**: 使用传统的未加密 Webhook 模式
- **Callback (加密)**: 使用新的加密回调模式，支持消息加密和签名验证

#### Webhook 模式参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| Webhook 地址 | x-agent 服务的 Webhook 端点 | `http://localhost:8080/api/wechat/webhook` |

#### Callback 模式参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| Callback 地址 | x-agent 服务的 Callback 端点 | `http://localhost:8080/api/wechat/callback` |
| Bot ID | 机器人的唯一标识 | `bot001` |
| Token | 企业微信回调 Token（必须与服务端配置一致） | - |
| EncodingAESKey | 企业微信消息加密密钥（43 位字符） | - |

#### 通用参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| 用户 ID (私聊) | 私聊用户的企业微信 ID | `user001` |
| 用户名 (私聊) | 私聊用户的显示名称 | `张三` |
| 群聊 ID | 群聊的唯一标识 | `wrAAAAAAABgQAAAAAAAAAA` |
| 用户 ID (群聊) | 群聊中发言用户的 ID | `user002` |
| 用户名 (群聊) | 群聊中发言用户的名称 | `李四` |
| 机器人名称 | 机器人的显示名称 | `x-agent` |

## 使用示例

### Callback 模式测试

#### 1. URL 验证测试

在 Callback 模式下，首先进行 URL 验证：

1. 在配置面板选择 **Callback (加密)** 模式
2. 填写 Token 和 EncodingAESKey（与服务端 `.env` 配置一致）
3. 点击 **测试 URL 验证** 按钮
4. 如果配置正确，会显示 "✓ URL 验证成功!"

#### 2. 加密消息测试

1. 确保 URL 验证通过
2. 在私聊或群聊面板输入消息
3. 消息会自动加密并发送到服务端
4. 服务端的加密响应会自动解密并显示

### Webhook 模式测试

#### 私聊测试

1. 在左侧私聊面板输入: `你好`
2. 点击"发送"按钮
3. 系统会发送 Webhook 请求到 x-agent
4. 机器人的回复会显示在聊天界面中

#### 群聊测试

1. 在中间群聊面板输入: `@x-agent 你好`
2. 点击"发送"按钮
3. 系统会发送群聊 Webhook 请求
4. 只有包含 @mention 的消息才会触发机器人回复

#### 命令测试

支持的命令：

- `/help` - 查看帮助信息
- `/new` - 创建新会话
- `/status` - 查看当前会话状态
- `/reset` - 重置当前会话

在私聊中直接输入命令，在群聊中需要加 `@x-agent` 前缀。

## 消息格式

### Callback 模式（加密）

#### URL 验证请求 (GET)

```
GET /api/wechat/callback/{botid}?msg_signature={signature}&timestamp={timestamp}&nonce={nonce}&echostr={encrypted_echo}
```

参数说明：
- `msg_signature`: SHA1(sort(token, timestamp, nonce, echostr))
- `timestamp`: Unix 时间戳
- `nonce`: 随机字符串
- `echostr`: AES 加密的随机字符串

响应：解密后的 echostr 原文

#### 加密消息请求 (POST)

```
POST /api/wechat/callback/{botid}?msg_signature={signature}&timestamp={timestamp}&nonce={nonce}
Content-Type: application/json

{
  "encrypt": "<encrypted_message>"
}
```

消息加密流程：
1. 构造原始消息 JSON（如 `{"msgtype":"text","text":{"content":"你好"}}`）
2. 使用 AES-256-CBC 加密（密钥由 EncodingAESKey 解码得到）
3. 计算签名：SHA1(sort(token, timestamp, nonce, encrypt))
4. 发送加密消息和签名

响应格式：
```json
{
  "encrypt": "<encrypted_response>",
  "msgsignature": "<signature>",
  "timestamp": "<timestamp>",
  "nonce": "<nonce>"
}
```

### Webhook 模式（未加密）

#### 私聊消息示例

```json
{
  "WebhookUrl": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=mock-private-key",
  "ChatId": "user001",
  "ChatType": "single",
  "From": {
    "UserId": "user001",
    "Name": "张三",
    "Alias": "user001"
  },
  "MsgId": "msg_1234567890_0",
  "MsgType": "text",
  "Text": {
    "Content": "你好"
  }
}
```

#### 群聊消息示例

```json
{
  "WebhookUrl": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=mock-group-key",
  "ChatId": "wrAAAAAAABgQAAAAAAAAAA",
  "ChatType": "group",
  "From": {
    "UserId": "user002",
    "Name": "李四",
    "Alias": "user002"
  },
  "MsgId": "msg_1234567890_1",
  "MsgType": "text",
  "Text": {
    "Content": "@x-agent 你好"
  }
}
```

## 响应处理

### Callback 模式

服务端返回加密的响应消息：

```json
{
  "encrypt": "<encrypted_stream_response>",
  "msgsignature": "<signature>",
  "timestamp": "<timestamp>",
  "nonce": "<nonce>"
}
```

解密后的消息格式（Stream 类型）：

```json
{
  "msgtype": "stream",
  "stream": {
    "id": "stream_1234567890",
    "finish": false,
    "content": "处理步骤 0: 已完成\n"
  }
}
```

Mock UI 会自动：
1. 验证响应签名
2. 解密响应消息
3. 解析 JSON 并显示在聊天界面

### Webhook 模式

x-agent 服务返回的响应格式：

```json
{
  "code": 0
}
```

由于 x-agent 采用异步处理模式，机器人的实际回复会通过 `WebhookUrl` 异步推送。在 Mock UI 中：

- Webhook 请求的响应状态会显示在右侧配置面板
- 系统消息会显示在聊天界面（表示请求成功/失败）
- 如果响应中包含 `text.content` 字段，会模拟显示机器人回复

## 故障排查

### 1. URL 验证失败（Callback 模式）

**现象**: 点击 "测试 URL 验证" 显示失败

**解决方案**:
- 确认 Token 和 EncodingAESKey 与服务端 `.env` 配置完全一致
- 检查 EncodingAESKey 长度是否为 43 位字符
- 验证服务端是否正确启动并监听 8080 端口
- 查看浏览器控制台和服务端日志

### 2. 消息加密/解密失败

**现象**: 发送消息后显示加密或解密错误

**解决方案**:
- 确认 Token 和 EncodingAESKey 配置正确
- 检查浏览器是否支持 CryptoJS 库（查看控制台是否有加载错误）
- 验证消息格式是否符合企业微信规范
- 查看服务端日志中的详细错误信息

### 3. Webhook 请求失败（Webhook 模式）

**现象**: 右侧面板显示红色错误信息

**解决方案**:
- 检查 x-agent 服务是否正常运行
- 确认 Webhook 地址配置正确
- 检查浏览器控制台的网络请求详情
- 验证是否存在 CORS 跨域问题

### 4. 没有收到机器人回复

**现象**: 发送消息后只看到系统提示，没有机器人回复

**原因**: x-agent 采用异步处理，机器人回复需要通过 WebhookUrl 推送

**说明**: 
- Mock UI 中的 WebhookUrl 是模拟地址
- 实际的机器人回复会在 x-agent 服务日志中看到
- 可以查看 x-agent 的控制台输出来确认处理流程

### 5. 群聊消息不响应

**现象**: 群聊发送消息后没有任何反应

**解决方案**:
- 确保消息中包含 `@x-agent`
- 检查 x-agent 日志确认是否收到 Webhook
- 验证 x-agent 的 @mention 检测逻辑

## 技术实现

### 技术栈

- **Vue 3**: 响应式 UI 框架
- **CryptoJS**: AES-256-CBC 加密/解密和 SHA1 签名
- **原生 JavaScript**: 无需编译，直接运行
- **Fetch API**: 发送 HTTP 请求
- **LocalStorage**: 配置持久化
- **CSS3**: 响应式布局和动画

### 架构设计

#### Callback 模式流程

```
用户输入
    ↓
Vue 组件处理
    ↓
构建原始消息 JSON
    ↓
AES-256-CBC 加密
    ↓
计算 SHA1 签名
    ↓
发送 POST /api/wechat/callback/:botid
    ↓
x-agent 验证签名 + 解密
    ↓
处理消息 + 生成响应
    ↓
加密响应 + 签名
    ↓
Mock UI 验证 + 解密
    ↓
UI 更新显示
```

#### Webhook 模式流程

```
用户输入
    ↓
Vue 组件处理
    ↓
构建 Webhook 消息
    ↓
发送 HTTP POST 请求
    ↓
x-agent Webhook 端点
    ↓
异步处理 + 响应
    ↓
UI 更新显示
```

### 特性说明

1. **单文件设计**: 所有代码在一个 HTML 文件中，无需构建工具
2. **CDN 加载**: 从 CDN 加载 Vue 3 和 CryptoJS，无需本地依赖
3. **响应式数据**: 使用 Vue 3 Options API 管理状态
4. **自动滚动**: 新消息自动滚动到底部
5. **时间戳**: 每条消息带有发送时间
6. **消息 ID**: 自动生成唯一消息 ID
7. **加密实现**: 完整的 AES-256-CBC 加密/解密和 SHA1 签名验证
8. **配置持久化**: 使用 LocalStorage 保存 Token 和密钥配置

## 扩展和定制

### 添加新的消息类型

#### Callback 模式

在 `sendCallbackMessage` 方法中修改消息结构：

```javascript
const message = {
  msgtype: 'image',  // 改为其他类型
  image: {
    url: 'https://example.com/image.jpg'
  }
};
```

#### Webhook 模式

在 `sendWebhookRequest` 方法中修改 payload 结构：

```javascript
const webhookPayload = {
  // ... 现有字段
  MsgType: 'image',
  Image: {
    MediaId: 'xxx'
  }
};
```

### 自定义样式

修改 `<style>` 标签中的 CSS：

```css
.message.user .message-content {
  background: #your-color;
}
```

### 添加更多配置项

在 `data()` 中添加新字段，并在模板中添加对应的 input 元素。

## 环境变量说明

在 `.env` 文件中配置以下变量：

```bash
# 必需（Callback 模式）
WECHAT_TOKEN=your_secure_token
WECHAT_ENCODING_AES_KEY=your_43_char_base64_encoded_key

# 可选
SERVER_PORT=8080              # 服务端口
REDIS_HOST=localhost:6379     # Redis 地址
```

### 生成 EncodingAESKey

可以使用以下方法生成 43 位的 EncodingAESKey：

```bash
# 方法 1: 使用 openssl
openssl rand -base64 32 | cut -c1-43

# 方法 2: 使用 Python
python3 -c "import base64, os; print(base64.b64encode(os.urandom(32)).decode()[:43])"

# 方法 3: 在企业微信后台自动生成
```

## 参考文档

- [企业微信智能机器人 API 文档](https://developer.work.weixin.qq.com/document/path/101039)
- [企业微信消息加密解密技术方案](https://developer.work.weixin.qq.com/document/path/90968)
- [Vue 3 官方文档](https://v3.vuejs.org/)
- [CryptoJS 文档](https://cryptojs.gitbook.io/docs/)
- [x-agent 企业微信集成文档](./wechat-integration.md)

## 常见问题

**Q: 为什么使用单文件 HTML 而不是标准的 Vue 项目?**

A: 为了简化部署和使用。用户只需在浏览器中打开文件即可使用，无需安装 Node.js、npm 或执行构建流程。

**Q: 可以部署到生产环境吗?**

A: 这是一个 Mock 测试工具，仅用于本地开发测试。生产环境应该使用真实的企业微信接入。

**Q: Callback 模式和 Webhook 模式有什么区别?**

A: 
- **Callback 模式**: 使用 AES-256-CBC 加密和 SHA1 签名，符合企业微信官方加密规范，更安全
- **Webhook 模式**: 传统的未加密模式，用于兼容旧版本或简单测试

**Q: Token 和 EncodingAESKey 在哪里获取?**

A: 
1. 企业微信管理后台自动生成
2. 或使用上述方法自行生成
3. 确保 Mock UI 和服务端配置完全一致

**Q: CORS 跨域问题怎么解决?**

A: 
1. 使用同源的 HTTP 服务器托管 HTML 文件
2. x-agent 已内置 CORS 支持（见 `cmd/server/main.go:60-71`）
3. 如仍有问题，检查浏览器控制台的详细错误信息

## API 端点

x-agent 提供以下企业微信相关端点：

### Callback 端点（加密模式）

- `GET /api/wechat/callback/:botid` - URL 验证
- `POST /api/wechat/callback/:botid` - 加密消息回调

### Webhook 端点（传统模式）

- `POST /api/wechat/webhook` - 未加密消息 Webhook

## License

MIT
