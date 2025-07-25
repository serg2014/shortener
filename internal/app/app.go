package app

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/serg2014/shortener/internal/config"
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

func GenerateShortURL(ctx context.Context, store storage.Storager, origURL string) (string, error) {
	shortURL := generateShortKey()
	err := store.Set(ctx, shortURL, origURL)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			return "", err
		}
		shortURL, ok, err := store.GetShort(ctx, origURL)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("can not find origurl %s", origURL)
		}
		return URLTemplate(shortURL), storage.ErrConflict
	}

	return URLTemplate(shortURL), nil
}

func URLTemplate(id string) string {
	return fmt.Sprintf("%s%s", config.Config.URL(), id)
}

func GenerateShortURLBatch(ctx context.Context, store storage.Storager, req models.RequestBatch) (models.ResponseBatch, error) {
	resp := make(models.ResponseBatch, len(req))
	short2orig := make(map[string]string, len(req))
	for i := range req {
		resp[i].CorrelationID = req[i].CorrelationID
		id := generateShortKey()
		resp[i].ShortURL = URLTemplate(id)
		short2orig[id] = req[i].OriginalURL
	}

	err := store.SetBatch(ctx, short2orig)
	return resp, err
}
