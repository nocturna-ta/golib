package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const (
	keySize = 32
)

var (
	ErrEmptyPlaintext = errors.New("plaintext is empty")
	ErrInvalidKey     = errors.New("invalid key")
)

func GenerateKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, errors.New("failed to generate key: " + err.Error())
	}
	return key, nil
}

func GenerateKeyString() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}
func ParseKeyString(keyStr string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, errors.New("failed to decode key: " + err.Error())
	}
	if len(key) != keySize {
		return nil, ErrInvalidKey
	}
	return key, nil
}
