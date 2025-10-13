package app

import (
	"math/rand"

	"github.com/serg2014/shortener/internal/storage"
)

type Generate struct{}

func (g *Generate) GenerateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	shortKey := make([]byte, storage.KeyLength)
	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortKey)
}

type Genarator interface {
	GenerateShortKey() string
}
