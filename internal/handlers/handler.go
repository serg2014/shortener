package handlers

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/storage"
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

		shortURL := generateShortKey()
		store.Set(shortURL, string(origURL))
		body := urlTemplate(shortURL)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(body))
	}
}

func urlTemplate(id string) string {
	return fmt.Sprintf("%s%s", config.NewConfig.URL(), id)
}

func generateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 8

	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortKey)
}

func GetURL(store storage.Storager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "key")
		origURL, err := getOrigURL(store, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

func Router(store storage.Storager) chi.Router {
	r := chi.NewRouter()
	r.Post("/", CreateURL(store))  // POST /
	r.Get("/{key}", GetURL(store)) // GET /Fvdvgfgf
	return r
}
