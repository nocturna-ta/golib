package encryption

import (
	"encoding/base64"
	"errors"
)

const (
	keySize = 32
)

var (
	ErrEmptyPlaintext = errors.New("plaintext is empty")
	ErrInvalidKey     = errors.New("invalid key")
)

type Encryption struct {
	key []byte
}

func NewEncryption(encodedKey string) (*Encryption, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, errors.New("failed to decode key")
	}

	if len(key) != keySize {
		return nil, ErrInvalidKey
	}

	return &Encryption{key: key}, nil

}
