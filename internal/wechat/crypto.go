package wechat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Crypto struct {
	token          string
	encodingAESKey string
	corpID         string
	aesKey         []byte
}

func NewCrypto(token, encodingAESKey, corpID string) (*Crypto, error) {
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("failed to decode aes key: %w", err)
	}

	return &Crypto{
		token:          token,
		encodingAESKey: encodingAESKey,
		corpID:         corpID,
		aesKey:         aesKey,
	}, nil
}

func (c *Crypto) VerifySignature(msgSignature, timestamp, nonce, msgEncrypt string) bool {
	signature := c.calculateSignature(timestamp, nonce, msgEncrypt)
	return signature == msgSignature
}

func (c *Crypto) calculateSignature(timestamp, nonce, msgEncrypt string) string {
	strs := []string{c.token, timestamp, nonce, msgEncrypt}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Crypto) DecryptMsg(msgSignature, timestamp, nonce, msgEncrypt string) (string, error) {
	if !c.VerifySignature(msgSignature, timestamp, nonce, msgEncrypt) {
		return "", errors.New("signature verification failed")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(msgEncrypt)
	if err != nil {
		return "", fmt.Errorf("failed to decode message: %w", err)
	}

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := c.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	plaintext = pkcs7Unpad(plaintext)

	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	msg := plaintext[20 : 20+msgLen]
	receivedCorpID := string(plaintext[20+msgLen:])

	if receivedCorpID != c.corpID {
		return "", errors.New("corp id mismatch")
	}

	return string(msg), nil
}

func (c *Crypto) EncryptMsg(msg, timestamp, nonce string) (string, error) {
	msgBytes := []byte(msg)
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	msgLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBuf, uint32(len(msgBytes)))

	plaintext := bytes.NewBuffer(randBytes)
	plaintext.Write(msgLenBuf)
	plaintext.Write(msgBytes)
	plaintext.WriteString(c.corpID)

	plaintextBytes := pkcs7Pad(plaintext.Bytes(), aes.BlockSize)

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	iv := c.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)

	ciphertext := make([]byte, len(plaintextBytes))
	mode.CryptBlocks(ciphertext, plaintextBytes)

	msgEncrypt := base64.StdEncoding.EncodeToString(ciphertext)

	return msgEncrypt, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	unpadding := int(data[length-1])
	if unpadding > length {
		return data
	}
	return data[:(length - unpadding)]
}
