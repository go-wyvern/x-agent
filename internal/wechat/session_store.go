package wechat

import (
	"sync"
)

type UserSessionStore struct {
	mapping map[string]string
	mutex   sync.RWMutex
}

func NewUserSessionStore() *UserSessionStore {
	return &UserSessionStore{
		mapping: make(map[string]string),
	}
}

func (uss *UserSessionStore) GetSession(userID string) string {
	uss.mutex.RLock()
	defer uss.mutex.RUnlock()
	return uss.mapping[userID]
}

func (uss *UserSessionStore) SetSession(userID, sessionID string) {
	uss.mutex.Lock()
	defer uss.mutex.Unlock()
	uss.mapping[userID] = sessionID
}

func (uss *UserSessionStore) ClearSession(userID string) {
	uss.mutex.Lock()
	defer uss.mutex.Unlock()
	delete(uss.mapping, userID)
}
