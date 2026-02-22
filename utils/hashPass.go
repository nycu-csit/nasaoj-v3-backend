package utils

import (
	"crypto/sha256"
	"encoding/base64"
)

func HashPass(pass string) string {
	sum := sha256.Sum256([]byte(pass))
	return base64.URLEncoding.EncodeToString(sum[:])
}
