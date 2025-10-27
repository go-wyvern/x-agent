package container

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Config struct {
	ImageTag                    string
	AnthropicBaseURL            string
	AnthropicAuthToken          string
	AnthropicModel              string
	AnthropicDefaultSonnetModel string
	AnthropicDefaultHaikuModel  string
	AnthropicSmallFastModel     string
}

type Manager struct {
	client     *client.Client
	containers map[string]*Container
	mutex      sync.RWMutex
	config     *Config
}

func NewManager(config *Config) (*Manager, error) {
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
		config:     config,
	}, nil
}

func (m *Manager) CreateContainer(sessionID, workspacePath string) (*Container, error) {
	containerName := fmt.Sprintf("xagent-session-%s", sessionID)

	config := &container.Config{
		Image: m.config.ImageTag,
		Cmd:   []string{"tail", "-f", "/dev/null"},
		Tty:   true,
		Env: []string{
			fmt.Sprintf("ANTHROPIC_BASE_URL=%s", m.config.AnthropicBaseURL),
			fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", m.config.AnthropicAuthToken),
			fmt.Sprintf("ANTHROPIC_MODEL=%s", m.config.AnthropicModel),
			fmt.Sprintf("ANTHROPIC_DEFAULT_SONNET_MODEL=%s", m.config.AnthropicDefaultSonnetModel),
			fmt.Sprintf("ANTHROPIC_DEFAULT_HAIKU_MODEL=%s", m.config.AnthropicDefaultHaikuModel),
			fmt.Sprintf("ANTHROPIC_SMALL_FAST_MODEL=%s", m.config.AnthropicSmallFastModel),
		},
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
		manager:       m,
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
