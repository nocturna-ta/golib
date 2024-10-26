package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func Encrypt(plaintext string, key []byte) (string, error) {
	if len(plaintext) == 0 {
		return "", ErrEmptyPlaintext
	}
	if len(key) != keySize {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.New("error creating AES cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("error creating GCM")
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.New("error generating random nonce")
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}
