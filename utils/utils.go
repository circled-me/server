package utils

import (
	"crypto/sha512"
	"encoding/hex"
)

// Sha512String hashes and encodes in hex the result
func Sha512String(s string) string {
	hash := sha512.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}
