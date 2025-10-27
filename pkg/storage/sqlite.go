package storage

import (
	"encoding/json"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/go-wyvern/x-agent/internal/models"
)

type SQLiteSessionStore struct {
	db *gorm.DB
}

type sessionRecord struct {
	ID            string `gorm:"primaryKey"`
	UserID        string
	WorkspacePath string
	ContainerID   string
	MessagesJSON  string
	MetadataJSON  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
	Status        int
}

func NewSQLiteSessionStore(dbPath string) (*SQLiteSessionStore, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&sessionRecord{}); err != nil {
		return nil, err
	}

	return &SQLiteSessionStore{db: db}, nil
}

func (s *SQLiteSessionStore) Save(session *models.Session) error {
	messagesJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return err
	}

	record := &sessionRecord{
		ID:            session.ID,
		UserID:        session.UserID,
		WorkspacePath: session.WorkspacePath,
		ContainerID:   session.ContainerID,
		MessagesJSON:  string(messagesJSON),
		MetadataJSON:  string(metadataJSON),
		CreatedAt:     session.CreatedAt,
		UpdatedAt:     session.UpdatedAt,
		ExpiresAt:     session.ExpiresAt,
		Status:        int(session.Status),
	}

	return s.db.Save(record).Error
}

func (s *SQLiteSessionStore) Get(sessionID string) (*models.Session, error) {
	var record sessionRecord
	err := s.db.Where("id = ?", sessionID).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	var messages []models.Message
	if err := json.Unmarshal([]byte(record.MessagesJSON), &messages); err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(record.MetadataJSON), &metadata); err != nil {
		return nil, err
	}

	session := &models.Session{
		ID:            record.ID,
		UserID:        record.UserID,
		WorkspacePath: record.WorkspacePath,
		ContainerID:   record.ContainerID,
		Messages:      messages,
		Metadata:      metadata,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
		ExpiresAt:     record.ExpiresAt,
		Status:        models.SessionStatus(record.Status),
	}

	return session, nil
}

func (s *SQLiteSessionStore) Delete(sessionID string) error {
	return s.db.Where("id = ?", sessionID).Delete(&sessionRecord{}).Error
}

func (s *SQLiteSessionStore) FindExpired() ([]*models.Session, error) {
	var records []sessionRecord
	now := time.Now()
	err := s.db.Where("expires_at < ?", now).Find(&records).Error
	if err != nil {
		return nil, err
	}

	sessions := make([]*models.Session, 0, len(records))
	for _, record := range records {
		var messages []models.Message
		if err := json.Unmarshal([]byte(record.MessagesJSON), &messages); err != nil {
			continue
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(record.MetadataJSON), &metadata); err != nil {
			continue
		}

		session := &models.Session{
			ID:            record.ID,
			UserID:        record.UserID,
			WorkspacePath: record.WorkspacePath,
			ContainerID:   record.ContainerID,
			Messages:      messages,
			Metadata:      metadata,
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
			ExpiresAt:     record.ExpiresAt,
			Status:        models.SessionStatus(record.Status),
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
