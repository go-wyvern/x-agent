# X-Agent - Docker Container Management System

Docker container management system for Claude-code integration with session-based isolation.

## Features

- **Session Isolation**: Each session gets its own Docker container
- **Resource Management**: CPU and memory limits for containers
- **Auto Cleanup**: Automatic cleanup of idle containers
- **Claude-code Integration**: Execute claude-code commands in containers
- **Container Lifecycle**: Full lifecycle management (create, start, stop, status)
- **Logging**: Container log collection and monitoring

## Architecture

### Components

- **Container Manager**: Core component managing Docker containers
- **Container Types**: Type definitions for container states and metadata
- **Docker Integration**: Uses Docker SDK for Go

### Container Lifecycle

1. Create container for session
2. Mount workspace directory
3. Execute claude-code commands
4. Monitor container status
5. Auto cleanup after idle timeout

## Usage

### Building Docker Image

```bash
docker build -t xagent:latest .
```

### Using Container Manager

```go
import "github.com/go-wyvern/x-agent/pkg/container"

// Create manager
manager, err := container.NewManager("xagent:latest")
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Start cleanup scheduler (check every 30 min, cleanup after 2 hours idle)
manager.StartCleanupScheduler(30*time.Minute, 2*time.Hour)

// Create container for session
c, err := manager.CreateContainer("session-123", "/path/to/workspace")
if err != nil {
    log.Fatal(err)
}

// Execute claude-code
output, err := manager.ExecuteClaude("session-123", "Help me debug this code")
if err != nil {
    log.Fatal(err)
}
fmt.Println(output)

// Check status
status, err := manager.CheckContainerStatus("session-123")

// Get logs
logs, err := manager.GetContainerLogs("session-123", "100")

// Stop container
err = manager.StopContainer("session-123")
```

## Configuration

### Docker Image

The Dockerfile includes:
- Ubuntu 22.04 base
- Claude-code installation
- MCP configuration
- Required dependencies (curl, git, ca-certificates)

### MCP Configuration

Default MCP configuration provides:
- Filesystem server for `/workspace/knowledge` directory

### Resource Limits

Default limits per container:
- Memory: 4GB
- CPU: 2 cores

## API Reference

### Manager Methods

- `NewManager(imageTag string)`: Create new container manager
- `CreateContainer(sessionID, workspacePath string)`: Create and start container
- `GetContainer(sessionID string)`: Get container by session ID
- `StopContainer(sessionID string)`: Stop and remove container
- `CheckContainerStatus(sessionID string)`: Check container status
- `ExecuteClaude(sessionID, message string)`: Execute claude-code command
- `GetContainerLogs(sessionID, tail string)`: Get container logs
- `StartCleanupScheduler(interval, idleTimeout time.Duration)`: Start auto cleanup
- `Close()`: Stop all containers and close manager

### Container States

- `ContainerCreating`: Container is being created
- `ContainerRunning`: Container is running
- `ContainerStopped`: Container is stopped
- `ContainerError`: Container encountered an error

## Dependencies

- Issue #3: Git Worktree management (for workspace paths)
- Issue #5: Session management (for session IDs)

## Development

### Prerequisites

- Go 1.21+
- Docker Engine
- Docker SDK for Go

### Building

```bash
go mod download
go build ./...
```

### Testing

```bash
go test ./...
```

## License

MIT
