package randomizer

import (
	cryptoRand "crypto/rand"
	"math/big"
	"math/rand"
	"time"
	"unsafe"
)

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyz~!@#$%^&*()_+ABCDEFGHIJKLMNOPQRSTUVWXYZ`1234567890-="
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandomInt generates a cryptographically secure random integer between min and max (inclusive)
func RandomInt(min, max int64) (int64, error) {
	// Create a new big.Int to hold the range of possible values
	rangeSize := big.NewInt(max - min + 1)

	// Generate a cryptographically secure random value within the range
	randomValue, err := cryptoRand.Int(cryptoRand.Reader, rangeSize)
	if err != nil {
		return 0, err
	}

	// Convert the random value to an int64 and add the minimum value to shift the range
	return randomValue.Int64() + min, nil
}

// RandomString generates a cryptographically secure random string with a given length
func RandomString(length int) string {
	return RandomStringFromSeed(length, letterBytes)
}

// RandomStringFromSeed generates a cryptographically secure random string with a given length
func RandomStringFromSeed(length int, seed string) string {
	b := make([]byte, length)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := length-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(seed) {
			b[i] = seed[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}
