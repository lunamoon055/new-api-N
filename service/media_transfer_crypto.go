package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/common"
)

func mediaTransferKey() ([]byte, error) {
	// A random process-local secret would make queued jobs unreadable after a
	// restart. Fail closed until the deployment has a stable secret.
	if os.Getenv("CRYPTO_SECRET") == "" && os.Getenv("SESSION_SECRET") == "" {
		return nil, fmt.Errorf("media transfer requires a stable CRYPTO_SECRET or SESSION_SECRET")
	}
	sum := sha256.Sum256([]byte("new-api-media-transfer-v1:" + common.CryptoSecret))
	return sum[:], nil
}

func encryptMediaTransferValue(value string) (string, error) {
	key, err := mediaTransferKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decryptMediaTransferValue(value string) (string, error) {
	key, err := mediaTransferKey()
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("decode encrypted media transfer value: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < aead.NonceSize() {
		return "", fmt.Errorf("encrypted media transfer value is truncated")
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt media transfer value: %w", err)
	}
	return string(plain), nil
}
