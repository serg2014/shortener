package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/config"
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
		shortID := app.GenerateShortKey(store, string(origURL))
		body := UrlTemplate(shortID)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(body))
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

		shortID := app.GenerateShortKey(store, req.URL)
		resp := models.Response{
			Result: UrlTemplate(shortID),
		}

		// порядок важен
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// сериализуем ответ сервера
		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Debug("error encoding response", zap.Error(err))
			return
		}
	}
}

func UrlTemplate(id string) string {
	return fmt.Sprintf("%s%s", config.Config.URL(), id)
}

func GetURL(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "key")
		origURL, ok := store.Get(id)
		if !ok {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
	}
}

func getOrigURL(store storage.Storager, id string) (string, error) {
	if origURL, ok := store.Get(id); ok {
		return origURL, nil
	}
	return "", errors.New("bad id")
}
