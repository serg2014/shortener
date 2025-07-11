package app

import (
	"math/rand"

	"github.com/serg2014/shortener/internal/storage"
)

func generateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	shortKey := make([]byte, storage.KeyLength)
	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortKey)
}

func GenerateShortKey(store storage.Storager, origURL string) (string, error) {
	shortURL := generateShortKey()
	err := store.Set(shortURL, string(origURL))
	return shortURL, err
}
