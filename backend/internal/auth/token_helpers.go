package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func cryptoRandRead(b []byte) (int, error) {
	return rand.Read(b)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func base64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
