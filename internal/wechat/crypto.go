package wechat

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type Crypto struct {
	webhookKey string
}

func NewCrypto(webhookKey string) (*Crypto, error) {
	if webhookKey == "" {
		return nil, fmt.Errorf("webhook key cannot be empty")
	}

	return &Crypto{
		webhookKey: webhookKey,
	}, nil
}

func (c *Crypto) VerifyWebhook(signature, payload string) bool {
	expectedSignature := c.calculateSignature(payload)
	return signature == expectedSignature
}

func (c *Crypto) calculateSignature(payload string) string {
	h := sha256.New()
	h.Write([]byte(c.webhookKey + payload))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
