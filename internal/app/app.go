package app

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/models"
	"github.com/serg2014/shortener/internal/storage"
)

type MyApp struct {
	store storage.Storager
	// канал для отложенной отправки новых сообщений
	msgChan chan storage.Message
	gen     Genarator
}

func NewApp(store storage.Storager, gen Genarator) *MyApp {
	if gen == nil {
		gen = &Generate{}
	}
	app := &MyApp{
		store:   store,
		msgChan: make(chan storage.Message, 1024),
		gen:     gen,
	}
	return app
}

func (a *MyApp) GenerateShortURL(ctx context.Context, origURL string, userID auth.UserID) (string, error) {
	shortURL := a.gen.GenerateShortKey()
	err := a.store.Set(ctx, shortURL, origURL, string(userID))
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

func (a *MyApp) GenerateShortURLBatch(ctx context.Context, req models.RequestBatch, userID auth.UserID) (models.ResponseBatch, error) {
	resp := make(models.ResponseBatch, len(req))
	short2orig := make(map[string]string, len(req))
	for i := range req {
		resp[i].CorrelationID = req[i].CorrelationID
		id := a.gen.GenerateShortKey()
		resp[i].ShortURL = URLTemplate(id)
		short2orig[id] = req[i].OriginalURL
	}

	err := a.store.SetBatch(ctx, short2orig, string(userID))
	return resp, err
}

func (a *MyApp) GetUserURLS(ctx context.Context, userID auth.UserID) (models.ResponseUser, error) {
	data, err := a.store.GetUserURLS(ctx, string(userID))
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

func (a *MyApp) DeleteUserURLS(ctx context.Context, req models.RequestForDeleteURLS, userID auth.UserID) error {
	// TODO uniq input
	// TODO req может быть большим, можно побить на чанки
	a.msgChan <- storage.Message{
		UserID:   string(userID),
		ShortURL: req,
	}
	return nil
}

func (a *MyApp) DeleteUserURLsBackground(ctx context.Context) {
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

func (a *MyApp) Set(ctx context.Context, key, value string, userID auth.UserID) error {
	return a.store.Set(ctx, key, value, string(userID))
}

func (a *MyApp) Ping(ctx context.Context) error {
	return a.store.Ping(ctx)
}
