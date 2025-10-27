package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func decryptAESCBC(encryptedData, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encryptedData))
	mode.CryptBlocks(decrypted, encryptedData)

	decrypted, err = pkcs7UnpadImage(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unpad: %w", err)
	}

	return decrypted, nil
}

func pkcs7UnpadImage(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	padding := int(data[len(data)-1])

	if padding > 32 || padding == 0 {
		return nil, fmt.Errorf("invalid padding: %d", padding)
	}

	if padding > len(data) {
		return nil, fmt.Errorf("padding size exceeds data length")
	}

	return data[:len(data)-padding], nil
}
