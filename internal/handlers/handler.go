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

func createURL(ctx context.Context, a *app.MyApp, origURL string, userID auth.UserID, w http.ResponseWriter) (int, string, error) {
	if origURL == "" {
		logger.Log.Debug("empty url")
		http.Error(w, "empty url", http.StatusBadRequest)
		return 0, "", errors.New("empty url")
	}
	status := http.StatusCreated
	shortURL, err := a.GenerateShortURL(ctx, origURL, userID)
	if err != nil {
		if !errors.Is(err, storage.ErrConflict) {
			logger.Log.Error("can not generate short", zap.Error(err))
			code := http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			return 0, "", err
		}
		status = http.StatusConflict
	}
	return status, shortURL, nil
}

func noUser(w http.ResponseWriter, err error) {
	logger.Log.Error("can not find userid", zap.Error(err))
	http.Error(w, "no user", http.StatusInternalServerError)
}

func CreateURL(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Error("bad input", zap.Error(err))
			http.Error(w, "bad input", http.StatusBadRequest)
			return
		}
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			noUser(w, err)
			return
		}
		status, shortURL, err := createURL(r.Context(), a, string(origURL), userID, w)
		if err != nil {
			// ошибка обработа в createURL и клиенту уже отправили ответ
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(shortURL))
	}
}

// TODO copy paste
func CreateURLJson(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.Request
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			noUser(w, err)
			return
		}
		status, shortURL, err := createURL(r.Context(), a, req.URL, userID, w)
		if err != nil {
			// ошибка обработа в createURL и клиенту уже отправили ответ
			return
		}
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

func GetURL(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "key")
		origURL, ok, err := a.Get(r.Context(), id)
		var code int
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrDeleted):
				code = http.StatusGone
			default:
				code = http.StatusInternalServerError
			}
			logger.Log.Error("error in a.Get", zap.Error(err))
			http.Error(w, http.StatusText(code), code)
			return
		}
		if !ok {
			http.Error(w, "bad short url", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
	}
}

func Ping(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if err := a.Ping(ctx); err != nil {
			logger.Log.Error("ping", zap.Error(err))
			code := http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
		}

	}
}

// TODO copy paste func CreateURL2
func CreateURLBatch(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			noUser(w, err)
			return
		}

		var req models.RequestBatch
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		// TODO проверить что прислали урл. correlation_id должен быть уникальным
		resp, err := a.GenerateShortURLBatch(r.Context(), req, userID)
		if err != nil {
			logger.Log.Error(
				"can not generate short batch",
				zap.Error(err),
			)
			code := http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
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

func GetUserURLS(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			noUser(w, err)
			return
		}
		data, err := a.GetUserURLS(r.Context(), userID)
		if err != nil {
			logger.Log.Error("GetUserURLS", zap.Error(err))
			code := http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			return
		}
		if len(data) == 0 {
			code := http.StatusNoContent
			http.Error(w, http.StatusText(code), code)
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

func DeleteUserURLS(a *app.MyApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			noUser(w, err)
			return
		}

		var req models.RequestForDeleteURLS
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			code := http.StatusBadRequest
			http.Error(w, http.StatusText(code), code)
			return
		}

		err = a.DeleteUserURLS(r.Context(), req, userID)
		if err != nil {
			logger.Log.Error("DeleteUserURLS", zap.Error(err))
			code := http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}
