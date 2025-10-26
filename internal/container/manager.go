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

type ContainerStatus int

const (
	ContainerCreating ContainerStatus = iota
	ContainerRunning
	ContainerStopped
	ContainerError
)

type Container struct {
	SessionID     string
	ContainerID   string
	ContainerName string
	WorkspacePath string
	Status        ContainerStatus
	CreatedAt     time.Time
	LastUsedAt    time.Time
}

type ContainerManager struct {
	client     *client.Client
	containers map[string]*Container
	mutex      sync.RWMutex
	imageTag   string
}

func NewContainerManager(imageTag string) (*ContainerManager, error) {
	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	cm := &ContainerManager{
		client:     dockerClient,
		containers: make(map[string]*Container),
		imageTag:   imageTag,
	}

	cm.StartCleanupScheduler()

	return cm, nil
}

func (cm *ContainerManager) CreateContainer(sessionID, workspacePath string) (*Container, error) {
	containerName := fmt.Sprintf("xagent-session-%s", sessionID)

	config := &container.Config{
		Image: cm.imageTag,
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

	resp, err := cm.client.ContainerCreate(
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

	err = cm.client.ContainerStart(context.Background(), resp.ID, container.StartOptions{})
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

	cm.mutex.Lock()
	cm.containers[sessionID] = c
	cm.mutex.Unlock()

	log.Printf("Created container %s for session %s", containerName, sessionID)

	return c, nil
}

func (cm *ContainerManager) GetContainer(sessionID string) (*Container, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	container, exists := cm.containers[sessionID]
	if !exists {
		return nil, fmt.Errorf("container not found for session %s", sessionID)
	}

	return container, nil
}

func (cm *ContainerManager) ExecuteClaude(sessionID, message string) (string, error) {
	cm.mutex.RLock()
	c := cm.containers[sessionID]
	cm.mutex.RUnlock()

	if c == nil {
		return "", fmt.Errorf("container not found for session %s", sessionID)
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

	execID, err := cm.client.ContainerExecCreate(
		context.Background(),
		c.ContainerID,
		execConfig,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := cm.client.ContainerExecAttach(
		context.Background(),
		execID.ID,
		container.ExecStartOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to attach exec: %w", err)
	}
	defer resp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output := &bytes.Buffer{}
	errOutput := &bytes.Buffer{}

	done := make(chan error, 1)
	go func() {
		_, err := stdcopy.StdCopy(output, errOutput, resp.Reader)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read output: %w", err)
		}
	case <-ctx.Done():
		return "", fmt.Errorf("command execution timeout")
	}

	c.LastUsedAt = time.Now()

	result := output.String()
	if errOutput.Len() > 0 {
		result += "\n" + errOutput.String()
	}

	return result, nil
}

func (cm *ContainerManager) StopContainer(sessionID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	c, exists := cm.containers[sessionID]
	if !exists {
		return fmt.Errorf("container not found for session %s", sessionID)
	}

	timeout := 10
	err := cm.client.ContainerStop(
		context.Background(),
		c.ContainerID,
		container.StopOptions{Timeout: &timeout},
	)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	c.Status = ContainerStopped
	delete(cm.containers, sessionID)

	log.Printf("Stopped container %s for session %s", c.ContainerName, sessionID)

	return nil
}

func (cm *ContainerManager) CheckContainerStatus(sessionID string) (ContainerStatus, error) {
	cm.mutex.RLock()
	c, exists := cm.containers[sessionID]
	cm.mutex.RUnlock()

	if !exists {
		return ContainerStopped, fmt.Errorf("container not found for session %s", sessionID)
	}

	containerJSON, err := cm.client.ContainerInspect(context.Background(), c.ContainerID)
	if err != nil {
		return ContainerError, fmt.Errorf("failed to inspect container: %w", err)
	}

	if !containerJSON.State.Running {
		cm.mutex.Lock()
		c.Status = ContainerStopped
		cm.mutex.Unlock()
		return ContainerStopped, nil
	}

	return ContainerRunning, nil
}

func (cm *ContainerManager) StartCleanupScheduler() {
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
			cm.cleanupIdleContainers()
		}
	}()
}

func (cm *ContainerManager) cleanupIdleContainers() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	idleTimeout := 2 * time.Hour
	now := time.Now()

	for sessionID, c := range cm.containers {
		if now.Sub(c.LastUsedAt) > idleTimeout {
			timeout := 10
			err := cm.client.ContainerStop(
				context.Background(),
				c.ContainerID,
				container.StopOptions{Timeout: &timeout},
			)
			if err != nil {
				log.Printf("Failed to stop idle container %s: %v", c.ContainerName, err)
				continue
			}

			delete(cm.containers, sessionID)
			log.Printf("Cleaned up idle container %s for session %s", c.ContainerName, sessionID)
		}
	}
}

func (cm *ContainerManager) GetContainerLogs(sessionID string, tail int) (string, error) {
	cm.mutex.RLock()
	c, exists := cm.containers[sessionID]
	cm.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("container not found for session %s", sessionID)
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	logs, err := cm.client.ContainerLogs(context.Background(), c.ContainerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	output := &bytes.Buffer{}
	_, err = stdcopy.StdCopy(output, output, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return output.String(), nil
}

func (cm *ContainerManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for sessionID := range cm.containers {
		timeout := 10
		err := cm.client.ContainerStop(
			context.Background(),
			cm.containers[sessionID].ContainerID,
			container.StopOptions{Timeout: &timeout},
		)
		if err != nil {
			log.Printf("Failed to stop container for session %s: %v", sessionID, err)
		}
	}

	return cm.client.Close()
}
