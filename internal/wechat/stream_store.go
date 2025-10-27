package wechat

import (
	"sync"
	"time"
)

type StreamData struct {
	SessionID string
	UserID    string
	Question  string
	Step      int
	MaxSteps  int
	CreatedAt time.Time
}

type StreamStore struct {
	mu      sync.RWMutex
	streams map[string]*StreamData
}

func NewStreamStore() *StreamStore {
	store := &StreamStore{
		streams: make(map[string]*StreamData),
	}
	go store.cleanupExpiredStreams()
	return store
}

func (s *StreamStore) SetStreamData(streamID string, data *StreamData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	s.streams[streamID] = data
}

func (s *StreamStore) GetStreamData(streamID string) *StreamData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.streams[streamID]
}

func (s *StreamStore) DeleteStreamData(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.streams, streamID)
}

func (s *StreamStore) cleanupExpiredStreams() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for streamID, data := range s.streams {
			if now.Sub(data.CreatedAt) > 30*time.Minute {
				delete(s.streams, streamID)
			}
		}
		s.mu.Unlock()
	}
}
