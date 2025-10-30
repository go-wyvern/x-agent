package session

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/go-wyvern/x-agent/internal/models"
	"github.com/go-wyvern/x-agent/pkg/storage"
	"github.com/go-wyvern/x-agent/pkg/workspace"
)

type SessionManager struct {
	store            storage.SessionStore
	mutex            sync.RWMutex
	sessionTTL       time.Duration
	skillsRepoURL    string
	skillsRepoBranch string
}

func NewSessionManager(store storage.SessionStore, ttl time.Duration, skillsRepoURL, skillsRepoBranch string) *SessionManager {
	if skillsRepoBranch == "" {
		skillsRepoBranch = "main"
	}
	return &SessionManager{
		store:            store,
		sessionTTL:       ttl,
		skillsRepoURL:    skillsRepoURL,
		skillsRepoBranch: skillsRepoBranch,
	}
}

func (sm *SessionManager) CreateSession(userID string) (*models.Session, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionID := uuid.New().String()

	workspacePath := ""
	if sm.skillsRepoURL != "" {
		var err error
		workspacePath, err = workspace.InitializeSessionWorkspace(workspace.SessionWorkspaceConfig{
			SessionID:        sessionID,
			SkillsRepoURL:    sm.skillsRepoURL,
			SkillsRepoBranch: sm.skillsRepoBranch,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize workspace: %w", err)
		}
	}

	session := &models.Session{
		ID:            sessionID,
		UserID:        userID,
		WorkspacePath: workspacePath,
		Messages:      []models.Message{},
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(sm.sessionTTL),
		Status:        models.SessionActive,
	}

	err := sm.store.Save(session)
	if err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (sm *SessionManager) GetSession(sessionID string) (*models.Session, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, err := sm.store.Get(sessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, storage.ErrSessionExpired
	}

	return session, nil
}

func (sm *SessionManager) UpdateSession(session *models.Session) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session.UpdatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(sm.sessionTTL)

	return sm.store.Save(session)
}

func (sm *SessionManager) AddMessage(sessionID string, role, content string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, err := sm.store.Get(sessionID)
	if err != nil {
		return err
	}

	if time.Now().After(session.ExpiresAt) {
		return storage.ErrSessionExpired
	}

	message := models.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, message)
	session.UpdatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(sm.sessionTTL)

	return sm.store.Save(session)
}

func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.store.Delete(sessionID)
}

func (sm *SessionManager) StartCleanupScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			sm.cleanupExpiredSessions()
		}
	}()
}

func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	expiredSessions, err := sm.store.FindExpired()
	if err != nil {
		log.Printf("Failed to find expired sessions: %v", err)
		return
	}

	for _, session := range expiredSessions {
		if err := sm.store.Delete(session.ID); err != nil {
			log.Printf("Failed to delete expired session %s: %v", session.ID, err)
			continue
		}
		log.Printf("Cleaned up expired session %s", session.ID)
	}
}

func (sm *SessionManager) SetWorkspaceAndContainer(sessionID, workspacePath, containerID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, err := sm.store.Get(sessionID)
	if err != nil {
		return err
	}

	session.WorkspacePath = workspacePath
	session.ContainerID = containerID
	session.UpdatedAt = time.Now()

	return sm.store.Save(session)
}
