package storage

import (
	"errors"

	"github.com/go-wyvern/x-agent/internal/models"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

type SessionStore interface {
	Save(session *models.Session) error
	Get(sessionID string) (*models.Session, error)
	Delete(sessionID string) error
	FindExpired() ([]*models.Session, error)
}
