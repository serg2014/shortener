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
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/models"
	"github.com/serg2014/shortener/internal/storage"
	"go.uber.org/zap"
)

// var store = storage.NewStorage(nil)

func createURL(ctx context.Context, store storage.Storager, origURL string, userID string, w http.ResponseWriter) (int, string, error) {
	if origURL == "" {
		logger.Log.Debug("empty url")
		http.Error(w, "empty url", http.StatusBadRequest)
		return 0, "", errors.New("empty url")
	}
	status := http.StatusCreated
	shortURL, err := app.GenerateShortURL(ctx, store, origURL, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			logger.Log.Error("can not generate short", zap.Error(err))
			http.Error(w, "", http.StatusInternalServerError)
			return 0, "", err
		}
		status = http.StatusConflict
	}
	return status, shortURL, nil
}

func CreateURL(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// TODO check err
		userID, _ := auth.GetUserID(w, r)
		logger.Log.Info("CreateURL", zap.String("userID", userID))
		status, shortURL, err := createURL(r.Context(), store, string(origURL), userID, w)
		if err != nil {
			// ошибка обработа в createURL и клиенту уже отправили ответ
			return
		}
		/*
			status := http.StatusCreated
			shortURL, err := app.GenerateShortURL(r.Context(), store, string(origURL))
			if err != nil {
				if !errors.Is(err, storage.ErrConflict) {
					logger.Log.Error("can not generate short", zap.Error(err))
					http.Error(w, "", http.StatusInternalServerError)
					return
				}
				status = http.StatusConflict
			}
		*/
		w.WriteHeader(status)
		w.Write([]byte(shortURL))
	}
}

// TODO copy paste
func CreateURLJson(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.Request
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// TODO check err
		userID, _ := auth.GetUserID(w, r)
		status, shortURL, err := createURL(r.Context(), store, req.URL, userID, w)
		if err != nil {
			// ошибка обработа в createURL и клиенту уже отправили ответ
			return
		}
		/*
			if req.URL == "" {
				logger.Log.Debug("empty url")
				http.Error(w, "empty url", http.StatusBadRequest)
				return
			}

			status := http.StatusCreated
			shortURL, err := app.GenerateShortURL(r.Context(), store, req.URL)
			if err != nil {
				if !errors.Is(err, storage.ErrConflict) {
					logger.Log.Error("can not generate short", zap.Error(err))
					http.Error(w, "", http.StatusInternalServerError)
					return
				}
				status = http.StatusConflict
			}
		*/
		resp := models.Response{
			Result: shortURL,
		}

		// порядок важен
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		// сериализуем ответ сервера
		// TODO в случае ошибки сериализации клиенту уже отдали статус 200ок
		// а тело будет битым. возможно стоит сначала сериализовать. данных мало поэтому кажется ок
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
		origURL, ok, err := store.Get(r.Context(), id)
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
		// TODO check err
		userID, _ := auth.GetUserID(w, r)

		var req models.RequestBatch
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// TODO проверить что прислали урл. correlation_id должен быть уникальным
		resp, err := app.GenerateShortURLBatch(r.Context(), store, req, userID)
		if err != nil {
			logger.Log.Error(
				"can not generate short batch",
				zap.Error(err),
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

func GetUserURLS(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO странный интерфейс
		userID, err := auth.GetUserIDFromCookie(r)
		logger.Log.Info("GetUserURLS", zap.String("userID", userID))
		if err != nil {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		/*
			userID, err := auth.GetUserID(w, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		*/
		data, err := app.GetUserURLS(r.Context(), store, userID)
		if err != nil {
			logger.Log.Error("GetUserURLS", zap.Error(err))
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		if len(data) == 0 {
			http.Error(w, "", http.StatusNoContent)
			return
		}

		// порядок важен
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// сериализуем ответ сервера
		enc := json.NewEncoder(w)
		if err := enc.Encode(data); err != nil {
			logger.Log.Error("error encoding response", zap.Error(err))
			return
		}
	}
}
