# X-Agent - AI Agent System with Session & Container Management

X-Agent 是一个基于 Claude Code 的 AI 代理系统,支持会话管理、工作区管理和容器化执行环境。

## Features / 功能特性

- ✅ **Session Management**: Complete session lifecycle management / 完整的会话生命周期管理
- ✅ **Multiple Storage Options**: Redis and SQLite support / 支持 Redis 和 SQLite
- ✅ **Session Persistence**: Automatic conversation history and context saving / 自动保存对话历史和上下文
- ✅ **Auto Cleanup**: TTL expiration and scheduled cleanup / TTL 过期机制和定时清理
- ✅ **OpenAI Compatible API**: Standard chat completion interface / OpenAI 兼容的聊天补全接口
- ✅ **Container Isolation**: Each session gets its own Docker container / 每个会话独立的 Docker 容器
- ✅ **Resource Management**: CPU and memory limits for containers / 容器资源限制
- ✅ **Claude-code Integration**: Execute claude-code commands in containers / 容器内执行 Claude-code 命令

## Quick Start / 快速开始

### Installation / 安装依赖

```bash
go mod download
```

### Using Redis Storage / 使用 Redis 存储

```bash
# Start Redis
redis-server

# Start server
go run cmd/server/main.go
```

### Using SQLite Storage / 使用 SQLite 存储

Modify `cmd/server/main.go`:

```go
// store := storage.NewRedisSessionStore("localhost:6379", 24*time.Hour)
store, _ := storage.NewSQLiteSessionStore("./sessions.db")
```

## API Usage / API 使用

### Create New Conversation / 创建新对话

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user123" \
  -d '{
    "model": "claude-code",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

Response will include `X-Session-ID` header for subsequent requests.

### Continue Conversation / 继续对话

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: <session-id>" \
  -H "X-User-ID: user123" \
  -d '{
    "model": "claude-code",
    "messages": [
      {"role": "user", "content": "继续对话"}
    ]
  }'
```

### Query Session / 查询 Session

```bash
curl http://localhost:8080/v1/sessions/<session-id>
```

### Delete Session / 删除 Session

```bash
curl -X DELETE http://localhost:8080/v1/sessions/<session-id>
```

## Project Structure / 项目结构

```
x-agent/
├── cmd/
│   └── server/           # Server entry point / 服务器入口
│       └── main.go
├── internal/
│   ├── api/              # API handlers / API 处理器
│   │   └── handler.go
│   ├── models/           # Data models / 数据模型
│   │   └── session.go
│   └── session/          # Session manager / Session 管理器
│       ├── manager.go
│       └── manager_test.go
└── pkg/
    ├── container/        # Container management / 容器管理
    └── storage/          # Storage layer / 存储层
        ├── interface.go
        ├── redis.go
        └── sqlite.go
```

## Configuration / 配置

### Session TTL

Default session expiration: 24 hours. Configurable when creating SessionManager:

```go
sessionManager := session.NewSessionManager(store, 48*time.Hour)
```

### Cleanup Scheduler / 清理调度

Default: runs every hour

```go
sessionManager.StartCleanupScheduler(2 * time.Hour)
```

### Docker Image

Build the Docker image:

```bash
docker build -t xagent:latest .
```

## Testing / 测试

```bash
go test ./...
```

## Dependencies / 依赖项

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - Web framework / Web 框架
- [redis/go-redis](https://github.com/redis/go-redis) - Redis client / Redis 客户端
- [gorm.io/gorm](https://gorm.io/) - ORM framework / ORM 框架
- [google/uuid](https://github.com/google/uuid) - UUID generation / UUID 生成
- [docker/docker](https://github.com/docker/docker) - Docker SDK / Docker SDK

## License / 许可证

MIT
