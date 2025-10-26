package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Manager struct {
	client     *client.Client
	containers map[string]*Container
	mutex      sync.RWMutex
	imageTag   string
}

func NewManager(imageTag string) (*Manager, error) {
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Manager{
		client:     dockerClient,
		containers: make(map[string]*Container),
		imageTag:   imageTag,
	}, nil
}

func (m *Manager) CreateContainer(sessionID, workspacePath string) (*Container, error) {
	containerName := fmt.Sprintf("xagent-session-%s", sessionID)

	config := &container.Config{
		Image: m.imageTag,
		Cmd:   []string{"tail", "-f", "/dev/null"},
		Tty:   true,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/workspace", workspacePath),
		},
		Resources: container.Resources{
			Memory:   4 * 1024 * 1024 * 1024,
			NanoCPUs: 2 * 1000000000,
		},
		AutoRemove: true,
	}

	resp, err := m.client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		nil,
		nil,
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	err = m.client.ContainerStart(context.Background(), resp.ID, container.StartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	c := &Container{
		SessionID:     sessionID,
		ContainerID:   resp.ID,
		ContainerName: containerName,
		WorkspacePath: workspacePath,
		Status:        ContainerRunning,
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
	}

	m.mutex.Lock()
	m.containers[sessionID] = c
	m.mutex.Unlock()

	log.Printf("Created container %s for session %s", containerName, sessionID)
	return c, nil
}

func (m *Manager) GetContainer(sessionID string) (*Container, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, exists := m.containers[sessionID]
	if !exists {
		return nil, fmt.Errorf("container not found for session %s", sessionID)
	}
	return c, nil
}

func (m *Manager) StopContainer(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, exists := m.containers[sessionID]
	if !exists {
		return fmt.Errorf("container not found for session %s", sessionID)
	}

	timeout := 10
	err := m.client.ContainerStop(context.Background(), c.ContainerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	c.Status = ContainerStopped
	delete(m.containers, sessionID)
	log.Printf("Stopped container %s for session %s", c.ContainerName, sessionID)
	return nil
}

func (m *Manager) CheckContainerStatus(sessionID string) (ContainerStatus, error) {
	m.mutex.RLock()
	c, exists := m.containers[sessionID]
	m.mutex.RUnlock()

	if !exists {
		return ContainerError, fmt.Errorf("container not found for session %s", sessionID)
	}

	inspect, err := m.client.ContainerInspect(context.Background(), c.ContainerID)
	if err != nil {
		c.Status = ContainerError
		return ContainerError, fmt.Errorf("failed to inspect container: %w", err)
	}

	if inspect.State.Running {
		c.Status = ContainerRunning
		return ContainerRunning, nil
	}

	c.Status = ContainerStopped
	return ContainerStopped, nil
}

func (m *Manager) Prompt(sessionID, message string) (io.ReadCloser, error) {
	m.mutex.RLock()
	c := m.containers[sessionID]
	m.mutex.RUnlock()

	if c == nil {
		return nil, fmt.Errorf("container not found for session %s", sessionID)
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

	execID, err := m.client.ContainerExecCreate(
		context.Background(),
		c.ContainerID,
		execConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := m.client.ContainerExecAttach(
		context.Background(),
		execID.ID,
		container.ExecStartOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}

	c.LastUsedAt = time.Now()
	log.Printf("Executed claude-code in container %s for session %s", c.ContainerName, sessionID)
	return resp.Conn, nil
}

func (m *Manager) StartCleanupScheduler(interval time.Duration, idleTimeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			m.cleanupIdleContainers(idleTimeout)
		}
	}()
	log.Printf("Started cleanup scheduler with interval %v and idle timeout %v", interval, idleTimeout)
}

func (m *Manager) cleanupIdleContainers(idleTimeout time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for sessionID, c := range m.containers {
		if now.Sub(c.LastUsedAt) > idleTimeout {
			timeout := 10
			err := m.client.ContainerStop(
				context.Background(),
				c.ContainerID,
				container.StopOptions{Timeout: &timeout},
			)
			if err != nil {
				log.Printf("Failed to stop idle container %s: %v", c.ContainerName, err)
				continue
			}

			delete(m.containers, sessionID)
			log.Printf("Cleaned up idle container %s for session %s", c.ContainerName, sessionID)
		}
	}
}

func (m *Manager) GetContainerLogs(sessionID string, tail string) (string, error) {
	m.mutex.RLock()
	c := m.containers[sessionID]
	m.mutex.RUnlock()

	if c == nil {
		return "", fmt.Errorf("container not found for session %s", sessionID)
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	}

	logs, err := m.client.ContainerLogs(context.Background(), c.ContainerID, options)
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

func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for sessionID := range m.containers {
		c := m.containers[sessionID]
		timeout := 10
		_ = m.client.ContainerStop(
			context.Background(),
			c.ContainerID,
			container.StopOptions{Timeout: &timeout},
		)
	}

	return m.client.Close()
}
