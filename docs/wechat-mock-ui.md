# 企业微信智能机器人 Mock UI 使用文档

## 概述

这是一个用于本地测试企业微信智能机器人 Webhook 回调的 Mock 前端界面，使用 Vue 3 + TypeScript 开发（单文件 HTML 实现）。

## 功能特性

- ✅ **双聊天界面**: 支持私聊和群聊两种模式
- ✅ **Webhook 请求**: 按照企业微信智能机器人规范发送 Webhook 请求
- ✅ **实时响应显示**: 在聊天界面中显示服务器响应内容
- ✅ **配置灵活**: 可自定义 Webhook 地址、用户信息等
- ✅ **快捷命令**: 支持 `/help`、`/new`、`/status`、`/reset` 等命令
- ✅ **@mention 支持**: 群聊模式自动处理 @x-agent 提及
- ✅ **响应状态显示**: 显示 Webhook 请求成功/失败状态

## 快速开始

### 1. 启动 x-agent 服务

首先确保 x-agent 服务正在运行：

```bash
cd /path/to/x-agent
go run cmd/server/main.go
```

服务默认监听在 `http://localhost:8080`

### 2. 打开 Mock UI

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

可配置的参数：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| Webhook 地址 | x-agent 服务的 Webhook 端点 | `http://localhost:8080/api/wechat/webhook` |
| 用户 ID (私聊) | 私聊用户的企业微信 ID | `user001` |
| 用户名 (私聊) | 私聊用户的显示名称 | `张三` |
| 群聊 ID | 群聊的唯一标识 | `wrAAAAAAABgQAAAAAAAAAA` |
| 用户 ID (群聊) | 群聊中发言用户的 ID | `user002` |
| 用户名 (群聊) | 群聊中发言用户的名称 | `李四` |
| 机器人名称 | 机器人的显示名称 | `x-agent` |

## 使用示例

### 私聊测试

1. 在左侧私聊面板输入: `你好`
2. 点击"发送"按钮
3. 系统会发送 Webhook 请求到 x-agent
4. 机器人的回复会显示在聊天界面中

### 群聊测试

1. 在中间群聊面板输入: `@x-agent 你好`
2. 点击"发送"按钮
3. 系统会发送群聊 Webhook 请求
4. 只有包含 @mention 的消息才会触发机器人回复

### 命令测试

支持的命令：

- `/help` - 查看帮助信息
- `/new` - 创建新会话
- `/status` - 查看当前会话状态
- `/reset` - 重置当前会话

在私聊中直接输入命令，在群聊中需要加 `@x-agent` 前缀。

## Webhook 消息格式

Mock UI 发送的 Webhook 请求格式符合企业微信智能机器人规范：

### 私聊消息示例

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

### 群聊消息示例

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

### 1. Webhook 请求失败

**现象**: 右侧面板显示红色错误信息

**解决方案**:
- 检查 x-agent 服务是否正常运行
- 确认 Webhook 地址配置正确
- 检查浏览器控制台的网络请求详情
- 验证是否存在 CORS 跨域问题

### 2. 没有收到机器人回复

**现象**: 发送消息后只看到系统提示，没有机器人回复

**原因**: x-agent 采用异步处理，机器人回复需要通过 WebhookUrl 推送

**说明**: 
- Mock UI 中的 WebhookUrl 是模拟地址
- 实际的机器人回复会在 x-agent 服务日志中看到
- 可以查看 x-agent 的控制台输出来确认处理流程

### 3. 群聊消息不响应

**现象**: 群聊发送消息后没有任何反应

**解决方案**:
- 确保消息中包含 `@x-agent`
- 检查 x-agent 日志确认是否收到 Webhook
- 验证 x-agent 的 @mention 检测逻辑

## 技术实现

### 技术栈

- **Vue 3**: 响应式 UI 框架
- **原生 JavaScript**: 无需编译，直接运行
- **Fetch API**: 发送 Webhook 请求
- **CSS3**: 响应式布局和动画

### 架构设计

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
2. **CDN 加载**: 从 CDN 加载 Vue 3，无需本地依赖
3. **响应式数据**: 使用 Vue 3 Composition API 管理状态
4. **自动滚动**: 新消息自动滚动到底部
5. **时间戳**: 每条消息带有发送时间
6. **消息 ID**: 自动生成唯一消息 ID

## 扩展和定制

### 添加新的消息类型

在 `sendWebhookRequest` 方法中修改 payload 结构：

```javascript
const webhookPayload = {
  // ... 现有字段
  MsgType: 'image', // 改为其他类型
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

## 参考文档

- [企业微信智能机器人 API 文档](https://developer.work.weixin.qq.com/document/path/100719)
- [Vue 3 官方文档](https://v3.vuejs.org/)
- [x-agent 企业微信集成文档](./wechat-integration.md)

## 常见问题

**Q: 为什么使用单文件 HTML 而不是标准的 Vue 项目?**

A: 为了简化部署和使用。用户只需在浏览器中打开文件即可使用，无需安装 Node.js、npm 或执行构建流程。

**Q: 可以部署到生产环境吗?**

A: 这是一个 Mock 测试工具，仅用于本地开发测试。生产环境应该使用真实的企业微信接入。

**Q: CORS 跨域问题怎么解决?**

A: 
1. 使用同源的 HTTP 服务器托管 HTML 文件
2. 在 x-agent 中配置 CORS 允许
3. 使用浏览器插件临时禁用 CORS（仅开发环境）

## License

MIT
