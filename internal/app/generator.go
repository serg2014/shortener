package app

import (
	"crypto/rand"
	"math/big"

	"github.com/serg2014/shortener/internal/storage"
)

// Generate
type Generate struct{}

// GenerateShortKey generate random string(use characters [a-zA-Z0-9]) with length of storage.KeyLength
func (g *Generate) GenerateShortKey() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	shortKey := make([]byte, storage.KeyLength)
	for i := range shortKey {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		shortKey[i] = charset[index.Int64()]
	}
	return string(shortKey), nil
}

// Generator iterface
type Generator interface {
	GenerateShortKey() (string, error)
}
