package app

import (
	"context"
	"math/rand"

	"github.com/serg2014/shortener/internal/models"
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

func GenerateShortKeyBatch(ctx context.Context, store storage.Storager, req models.RequestBatch) (models.ResponseBatch, error) {
	resp := make(models.ResponseBatch, len(req))
	short2orig := make(map[string]string, len(req))
	for i := range req {
		resp[i].CorrelationId = req[i].CorrelationId
		resp[i].ShortUrl = generateShortKey()
		short2orig[resp[i].ShortUrl] = req[i].OriginalURL
	}

	err := store.SetBatch(ctx, short2orig)
	return resp, err
}
