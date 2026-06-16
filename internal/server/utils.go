package server

import (
	"crypto/rand"
	"encoding/hex"
)

func randomCode() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
