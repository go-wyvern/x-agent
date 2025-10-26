# X-Agent

Docker container management system for x-agent with Claude-code integration.

## Features

- Docker container lifecycle management
- Claude-code integration via docker exec
- Automatic idle container cleanup
- Resource limits (CPU, memory)
- Container logs collection

## Project Structure

```
.
├── internal/
│   └── container/          # Container management package
│       ├── manager.go      # Container manager implementation
│       └── manager_test.go # Unit tests
├── go.mod
└── README.md
```

## Container Manager

The `ContainerManager` provides:

- **Container Creation**: Creates isolated Docker containers for each session
- **Claude Execution**: Executes claude-code commands in containers via `docker exec`
- **Status Monitoring**: Checks container health and status
- **Automatic Cleanup**: Removes idle containers after 2 hours
- **Resource Limits**: 4GB memory, 2 CPU cores per container
- **Log Collection**: Retrieves container logs for debugging

### Usage Example

```go
package main

import (
    "log"
    "github.com/go-wyvern/x-agent/internal/container"
)

func main() {
    cm, err := container.NewContainerManager("ubuntu:22.04")
    if err != nil {
        log.Fatal(err)
    }
    defer cm.Close()

    c, err := cm.CreateContainer("session-123", "/tmp/workspace")
    if err != nil {
        log.Fatal(err)
    }

    output, err := cm.ExecuteClaude("session-123", "help me write a function")
    if err != nil {
        log.Fatal(err)
    }

    log.Println(output)
}
```

## Testing

```bash
go test ./internal/container/... -v
```

## Dependencies

- Docker Engine
- Go 1.24.5+
- github.com/docker/docker v28.0.0+incompatible
