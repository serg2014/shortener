package app

import (
	"math/rand"

	"github.com/serg2014/shortener/internal/storage"
)

// Generate
type Generate struct{}

// GenerateShortKey generate random string(use characters [a-zA-Z0-9]) with length of storage.KeyLength
func (g *Generate) GenerateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	shortKey := make([]byte, storage.KeyLength)
	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortKey)
}

// Genarator iterface
type Genarator interface {
	GenerateShortKey() string
}
