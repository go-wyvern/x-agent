package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/go-wyvern/x-agent/internal/models"
)

type RedisSessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisSessionStore(addr string, ttl time.Duration) *RedisSessionStore {
	return &RedisSessionStore{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
		ttl: ttl,
	}
}

func (s *RedisSessionStore) Save(session *models.Session) error {
	key := "session:" + session.ID
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.client.Set(
		context.Background(),
		key,
		data,
		s.ttl,
	).Err()
}

func (s *RedisSessionStore) Get(sessionID string) (*models.Session, error) {
	key := "session:" + sessionID
	data, err := s.client.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	var session models.Session
	err = json.Unmarshal(data, &session)
	return &session, err
}

func (s *RedisSessionStore) Delete(sessionID string) error {
	key := "session:" + sessionID
	return s.client.Del(context.Background(), key).Err()
}

func (s *RedisSessionStore) FindExpired() ([]*models.Session, error) {
	ctx := context.Background()
	keys, err := s.client.Keys(ctx, "session:*").Result()
	if err != nil {
		return nil, err
	}

	var expiredSessions []*models.Session
	now := time.Now()

	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var session models.Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		if now.After(session.ExpiresAt) {
			expiredSessions = append(expiredSessions, &session)
		}
	}

	return expiredSessions, nil
}

func (s *RedisSessionStore) Close() error {
	return s.client.Close()
}
