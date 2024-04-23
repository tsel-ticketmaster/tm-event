package util

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// GenerateRandomHEX returns random hex number in string format by the given size (in bytes).
func GenerateRandomHEX(size int) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

// GenerateSecret returns hashed string from combination of plain and salt. It presented in base64 format. The used algorithm is sha512.
func GenerateSecret(plain, salt string, len int) string {
	b := pbkdf2.Key([]byte(plain), []byte(salt), 128, len, sha512.New)

	return base64.StdEncoding.EncodeToString(b)
}

func GenerateTimestampWithPrefix(prefix string) string {
	now := time.Now()
	micro := now.UnixNano()

	return fmt.Sprintf("%s%d", prefix, micro)
}

var (
	Uppercase        = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Lowercase        = "abcdefghijklmnopqrstuvwxyz"
	Numeric          = "0123456789"
	UppercaseNumeric = fmt.Sprintf("%s%s", Uppercase, Numeric)
	Alpha            = fmt.Sprintf("%s%s", Uppercase, Lowercase)
	AlphaNumeric     = fmt.Sprintf("%s%s", Alpha, Numeric)
)

func GenerateUniqueID(composition string, length int) (id string) {
	if len(composition) < 1 {
		return
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read error: %v", err))
	}
	for i, byt := range b {
		b[i] = composition[int(byt)%len(composition)]
	}
	return string(b)
}
