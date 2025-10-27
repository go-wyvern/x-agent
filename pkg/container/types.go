package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

type ContainerStatus int

const (
	ContainerCreating ContainerStatus = iota
	ContainerRunning
	ContainerStopped
	ContainerError
)

func (s ContainerStatus) String() string {
	switch s {
	case ContainerCreating:
		return "creating"
	case ContainerRunning:
		return "running"
	case ContainerStopped:
		return "stopped"
	case ContainerError:
		return "error"
	default:
		return "unknown"
	}
}

type Container struct {
	SessionID     string
	ContainerID   string
	ContainerName string
	WorkspacePath string
	Status        ContainerStatus
	CreatedAt     time.Time
	LastUsedAt    time.Time
	manager       *Manager
}

func (c *Container) Prompt(message string) (io.ReadCloser, error) {
	if c.manager == nil {
		return nil, fmt.Errorf("container manager is not set")
	}

	mcpConfigPath := "/etc/claude/mcp-config.json"
	cmd := []string{
		"claude",
		"--mcp-config", mcpConfigPath,
		"--dangerously-skip-permissions",
		"-c",
		"-p", message,
	}

	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	execID, err := c.manager.client.ContainerExecCreate(
		context.Background(),
		c.ContainerID,
		execConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := c.manager.client.ContainerExecAttach(
		context.Background(),
		execID.ID,
		container.ExecStartOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}

	c.LastUsedAt = time.Now()
	log.Printf("Executed claude-code in container %s for session %s", c.ContainerName, c.SessionID)
	return resp.Conn, nil
}

func (c *Container) Logs(tail string) (string, error) {
	if c.manager == nil {
		return "", fmt.Errorf("container manager is not set")
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	}

	logs, err := c.manager.client.ContainerLogs(context.Background(), c.ContainerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	buf := new(bytes.Buffer)
	_, err = stdcopy.StdCopy(buf, buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}
