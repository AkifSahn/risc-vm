package rest

import (
	"crypto/rand"
	"encoding/hex"
)

func generateId() string{
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

