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
	"fmt"
	"math/big"
	"sort"
	"strings"
)

type Crypto struct {
	token          string
	encodingAESKey string
	aesKey         []byte
}

func NewCrypto(token, encodingAESKey string) (*Crypto, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}
	if encodingAESKey == "" {
		return nil, fmt.Errorf("encoding AES key cannot be empty")
	}

	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	if len(aesKey) != 32 {
		return nil, fmt.Errorf("invalid AES key length: expected 32 bytes, got %d", len(aesKey))
	}

	return &Crypto{
		token:          token,
		encodingAESKey: encodingAESKey,
		aesKey:         aesKey,
	}, nil
}

func (c *Crypto) VerifyURL(msgSignature, timestamp, nonce, echoStr string) (string, error) {
	signature := c.calculateSignature(timestamp, nonce, echoStr)
	if signature != msgSignature {
		return "", fmt.Errorf("signature verification failed")
	}

	decrypted, err := c.decrypt(echoStr, "")
	if err != nil {
		return "", fmt.Errorf("failed to decrypt echostr: %w", err)
	}

	return decrypted, nil
}

func (c *Crypto) DecryptMsg(encryptedMsg, msgSignature, timestamp, nonce, receiveID string) (string, error) {
	signature := c.calculateSignature(timestamp, nonce, encryptedMsg)
	if signature != msgSignature {
		return "", fmt.Errorf("signature verification failed: expected %s, got %s", msgSignature, signature)
	}

	decrypted, err := c.decrypt(encryptedMsg, receiveID)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt message: %w", err)
	}

	return decrypted, nil
}

func (c *Crypto) EncryptMsg(replyMsg, nonce, timestamp string) (string, error) {
	encrypted, err := c.encrypt(replyMsg, "")
	if err != nil {
		return "", fmt.Errorf("failed to encrypt message: %w", err)
	}

	signature := c.calculateSignature(timestamp, nonce, encrypted)

	response := fmt.Sprintf(`{
    "encrypt": "%s",
    "msgsignature": "%s",
    "timestamp": "%s",
    "nonce": "%s"
}`, encrypted, signature, timestamp, nonce)

	return response, nil
}

func (c *Crypto) calculateSignature(timestamp, nonce, encrypt string) string {
	strs := []string{c.token, timestamp, nonce, encrypt}
	sort.Strings(strs)

	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Crypto) encrypt(text, receiveID string) (string, error) {
	randomStr := generateRandomString(16)

	textBytes := []byte(text)
	textLength := make([]byte, 4)
	binary.BigEndian.PutUint32(textLength, uint32(len(textBytes)))

	var buffer bytes.Buffer
	buffer.Write(randomStr)
	buffer.Write(textLength)
	buffer.Write(textBytes)
	buffer.Write([]byte(receiveID))

	plaintext := pkcs7Pad(buffer.Bytes(), aes.BlockSize)

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, len(plaintext))
	iv := c.aesKey[:aes.BlockSize]

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *Crypto) decrypt(encryptedText, receiveID string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of block size")
	}

	iv := c.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return "", err
	}

	if len(plaintext) < 20 {
		return "", fmt.Errorf("invalid plaintext length")
	}

	textLength := binary.BigEndian.Uint32(plaintext[16:20])
	if len(plaintext) < int(20+textLength) {
		return "", fmt.Errorf("invalid text length")
	}

	text := plaintext[20 : 20+textLength]

	if receiveID != "" {
		fromReceiveID := plaintext[20+textLength:]
		if string(fromReceiveID) != receiveID {
			return "", fmt.Errorf("receiveID mismatch: expected %s, got %s", receiveID, string(fromReceiveID))
		}
	}

	return string(text), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	if padding == 0 {
		padding = blockSize
	}
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	padding := int(data[len(data)-1])
	if padding < 1 || padding > 32 {
		padding = 0
	}

	if padding > len(data) {
		return nil, fmt.Errorf("invalid padding size")
	}

	return data[:len(data)-padding], nil
}

func generateRandomString(length int) []byte {
	const charset = "0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return result
}
