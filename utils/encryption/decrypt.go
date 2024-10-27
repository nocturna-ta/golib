package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

func (e *Encryption) Decrypt(encryptedString string) (string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(encryptedString)
	if err != nil {
		return "", errors.New("error decoding encrypted string")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", errors.New("error creating cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("error creating GCM")
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]

	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", errors.New("error decrypting ciphertext")
	}

	return string(plainText), nil

}
