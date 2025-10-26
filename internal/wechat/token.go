package wechat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type TokenManager struct {
	config    *Config
	token     string
	expiresAt time.Time
	mutex     sync.RWMutex
}

func NewTokenManager(config *Config) *TokenManager {
	return &TokenManager{
		config: config,
	}
}

func (tm *TokenManager) GetAccessToken() (string, error) {
	tm.mutex.RLock()
	if time.Now().Before(tm.expiresAt) {
		token := tm.token
		tm.mutex.RUnlock()
		return token, nil
	}
	tm.mutex.RUnlock()

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if time.Now().Before(tm.expiresAt) {
		return tm.token, nil
	}

	url := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		tm.config.CorpID,
		tm.config.Secret,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat api error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	tm.token = result.AccessToken
	tm.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return tm.token, nil
}
