package session

import (
	"testing"
	"time"

	"github.com/go-wyvern/x-agent/internal/models"
	"github.com/go-wyvern/x-agent/pkg/storage"
)

type mockStore struct {
	sessions map[string]*models.Session
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*models.Session),
	}
}

func (m *mockStore) Save(session *models.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockStore) Get(sessionID string) (*models.Session, error) {
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, storage.ErrSessionNotFound
	}
	return session, nil
}

func (m *mockStore) Delete(sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockStore) FindExpired() ([]*models.Session, error) {
	var expired []*models.Session
	now := time.Now()
	for _, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			expired = append(expired, session)
		}
	}
	return expired, nil
}

func TestCreateSession(t *testing.T) {
	store := newMockStore()
	manager := NewSessionManager(store, 24*time.Hour)

	session, err := manager.CreateSession("user123")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.UserID != "user123" {
		t.Errorf("Expected UserID to be 'user123', got '%s'", session.UserID)
	}

	if session.Status != models.SessionActive {
		t.Errorf("Expected Status to be Active, got %d", session.Status)
	}
}

func TestGetSession(t *testing.T) {
	store := newMockStore()
	manager := NewSessionManager(store, 24*time.Hour)

	created, _ := manager.CreateSession("user123")

	retrieved, err := manager.GetSession(created.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected session ID %s, got %s", created.ID, retrieved.ID)
	}
}

func TestAddMessage(t *testing.T) {
	store := newMockStore()
	manager := NewSessionManager(store, 24*time.Hour)

	session, _ := manager.CreateSession("user123")

	err := manager.AddMessage(session.ID, "user", "Hello")
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	retrieved, _ := manager.GetSession(session.ID)
	if len(retrieved.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(retrieved.Messages))
	}

	if retrieved.Messages[0].Content != "Hello" {
		t.Errorf("Expected message content 'Hello', got '%s'", retrieved.Messages[0].Content)
	}
}

func TestSessionExpiration(t *testing.T) {
	store := newMockStore()
	manager := NewSessionManager(store, 1*time.Millisecond)

	session, _ := manager.CreateSession("user123")

	time.Sleep(10 * time.Millisecond)

	_, err := manager.GetSession(session.ID)
	if err != storage.ErrSessionExpired {
		t.Errorf("Expected ErrSessionExpired, got %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	store := newMockStore()
	manager := NewSessionManager(store, 24*time.Hour)

	session, _ := manager.CreateSession("user123")

	err := manager.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	_, err = manager.GetSession(session.ID)
	if err != storage.ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}
