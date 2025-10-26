package models

import (
	"time"
)

type SessionStatus int

const (
	SessionActive SessionStatus = iota
	SessionIdle
	SessionExpired
)

type Session struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	WorkspacePath string                 `json:"workspace_path"`
	ContainerID   string                 `json:"container_id"`
	Messages      []Message              `json:"messages"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	Status        SessionStatus          `json:"status"`
}

type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
