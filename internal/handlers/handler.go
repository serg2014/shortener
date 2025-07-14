package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/models"
	"github.com/serg2014/shortener/internal/storage"
	"go.uber.org/zap"
)

//var store = storage.NewStorage(nil)

func CreateURL(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		shortURL, err := app.GenerateShortURL(store, string(origURL))
		if err != nil {
			logger.Log.Error("can not generate short", zap.String("error", err.Error()))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortURL))
	}
}

// TODO copy paste
func CreateURL2(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req models.Request
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			logger.Log.Debug("empty url")
			http.Error(w, "empty url", http.StatusBadRequest)
			return
		}

		shortURL, err := app.GenerateShortURL(store, req.URL)
		if err != nil {
			logger.Log.Error("can not generate short", zap.String("error", err.Error()))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		resp := models.Response{
			Result: shortURL,
		}

		// порядок важен
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// сериализуем ответ сервера
		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Error("error encoding response", zap.Error(err))
			return
		}
	}
}

func GetURL(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "key")
		origURL, ok, err := store.Get(id)
		if err != nil {
			logger.Log.Error("error in Get", zap.String("error", err.Error()))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
	}
}

func getOrigURL(store storage.Storager, id string) (string, error) {
	origURL, ok, err := store.Get(id)
	if err != nil {
		return "", err
	}
	if ok {
		return origURL, nil
	}

	return "", errors.New("bad id")
}

func Ping(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if err := store.Ping(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	}
}

// TODO copy paste func CreateURL2
func CreateURLBatch(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req models.RequestBatch
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// TODO проверить что прислали урл. correlation_id должен быть уникальным
		resp, err := app.GenerateShortURLBatch(r.Context(), store, req)
		if err != nil {
			logger.Log.Error(
				"can not generate short",
				zap.String("error", err.Error()),
			)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		// порядок важен
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// сериализуем ответ сервера
		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Error("error encoding response", zap.Error(err))
			return
		}
	}
}
