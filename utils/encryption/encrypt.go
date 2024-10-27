package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func (e *Encryption) Encrypt(plaintext string) (string, error) {
	if len(plaintext) == 0 {
		return "", ErrEmptyPlaintext
	}

	block, err := aes.NewCipher(e.key)
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
