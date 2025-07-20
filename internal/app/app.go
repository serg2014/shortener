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

func GenerateShortURL(ctx context.Context, store storage.Storager, origURL string, userID string) (string, error) {
	shortURL := generateShortKey()
	err := store.Set(ctx, shortURL, origURL, userID)
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

func GenerateShortURLBatch(ctx context.Context, store storage.Storager, req models.RequestBatch, userID string) (models.ResponseBatch, error) {
	resp := make(models.ResponseBatch, len(req))
	short2orig := make(map[string]string, len(req))
	for i := range req {
		resp[i].CorrelationID = req[i].CorrelationID
		id := generateShortKey()
		resp[i].ShortURL = URLTemplate(id)
		short2orig[id] = req[i].OriginalURL
	}

	err := store.SetBatch(ctx, short2orig, userID)
	return resp, err
}

func GetUserURLS(ctx context.Context, store storage.Storager, userID string) (models.ResponseUser, error) {
	data, err := store.GetUserURLS(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := make(models.ResponseUser, len(data))
	for i := range data {
		resp[i].ShortURL = URLTemplate(data[i].ShortURL)
		resp[i].OriginalURL = data[i].OriginalURL
	}
	return resp, nil
}
