package container

import (
	"testing"
	"time"
)

func TestContainerManager_New(t *testing.T) {
	cm, err := NewContainerManager("ubuntu:22.04")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cm.Close()

	if cm.imageTag != "ubuntu:22.04" {
		t.Errorf("Expected imageTag 'ubuntu:22.04', got '%s'", cm.imageTag)
	}

	if cm.containers == nil {
		t.Error("containers map should not be nil")
	}
}

func TestContainer_Status(t *testing.T) {
	statuses := []ContainerStatus{
		ContainerCreating,
		ContainerRunning,
		ContainerStopped,
		ContainerError,
	}

	for _, status := range statuses {
		c := &Container{
			SessionID:  "test-session",
			Status:     status,
			CreatedAt:  time.Now(),
			LastUsedAt: time.Now(),
		}

		if c.Status != status {
			t.Errorf("Expected status %v, got %v", status, c.Status)
		}
	}
}
