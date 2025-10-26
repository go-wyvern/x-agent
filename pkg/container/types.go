package container

import "time"

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
}
