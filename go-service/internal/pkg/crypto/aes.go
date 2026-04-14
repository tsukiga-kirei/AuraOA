package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// key 必须是 16/24/32 字节（AES-128/192/256）。
// 这里从配置中读取，由调用方保证长度。
var encryptionKey []byte

// SetKey 设置全局 AES 加密密钥，长度必须为 16、24 或 32 字节（对应 AES-128/192/256）。
func SetKey(key string) error {
	k := []byte(key)
	switch len(k) {
	case 16, 24, 32:
		encryptionKey = k
		return nil
	default:
		return errors.New("加密密钥长度必须为 16、24 或 32 字节")
	}
}

// Encrypt 使用 AES-GCM 加密明文，返回 base64 编码的密文。
// 若明文为空字符串，直接返回空字符串。
func Encrypt(plaintext string) (string, error) {
	if len(encryptionKey) == 0 {
		return "", errors.New("加密密钥未设置")
	}
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 base64 编码的 AES-GCM 密文，返回明文。
// 若密文为空字符串，直接返回空字符串。
func Decrypt(encoded string) (string, error) {
	if len(encryptionKey) == 0 {
		return "", errors.New("加密密钥未设置")
	}
	if encoded == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("密文长度不足，数据可能已损坏")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
