package app

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/models"
	"github.com/serg2014/shortener/internal/storage"
	"go.uber.org/zap"
)

type MyApp struct {
	store storage.Storager
	// канал для отложенной отправки новых сообщений
	msgChan chan storage.Message
}

func NewApp(store storage.Storager) *MyApp {
	app := &MyApp{
		store:   store,
		msgChan: make(chan storage.Message, 1024),
	}
	return app
}

func generateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	shortKey := make([]byte, storage.KeyLength)
	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortKey)
}

func (a *MyApp) GenerateShortURL(ctx context.Context, origURL string, userID string) (string, error) {
	shortURL := generateShortKey()
	err := a.store.Set(ctx, shortURL, origURL, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			return "", err
		}
		shortURL, ok, err := a.store.GetShort(ctx, origURL)
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

func (a *MyApp) GenerateShortURLBatch(ctx context.Context, req models.RequestBatch, userID string) (models.ResponseBatch, error) {
	resp := make(models.ResponseBatch, len(req))
	short2orig := make(map[string]string, len(req))
	for i := range req {
		resp[i].CorrelationID = req[i].CorrelationID
		id := generateShortKey()
		resp[i].ShortURL = URLTemplate(id)
		short2orig[id] = req[i].OriginalURL
	}

	err := a.store.SetBatch(ctx, short2orig, userID)
	return resp, err
}

func (a *MyApp) GetUserURLS(ctx context.Context, userID string) (models.ResponseUser, error) {
	data, err := a.store.GetUserURLS(ctx, userID)
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

func (a *MyApp) DeleteUserURLS(ctx context.Context, req models.RequestForDeleteURLS, userID string) error {
	// TODO uniq input
	// TODO req может быть большим, можно побить на чанки
	a.msgChan <- storage.Message{
		UserID:   userID,
		ShortURL: req,
	}
	return nil
}

func (a *MyApp) DeleteUserURLSBackground(ctx context.Context) {
	for {
		select {
		case mes := <-a.msgChan:
			err := a.store.DeleteUserURLS(ctx, mes.ShortURL, mes.UserID)
			if err != nil {
				logger.Log.Error("problem with DeleteUserURLS", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *MyApp) Get(ctx context.Context, id string) (string, bool, error) {
	return a.store.Get(ctx, id)
}

func (a *MyApp) Set(ctx context.Context, key, value, userID string) error {
	return a.store.Set(ctx, key, value, userID)
}

func (a *MyApp) Ping(ctx context.Context) error {
	return a.store.Ping(ctx)
}
