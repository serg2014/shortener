package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serg2014/shortener/internal/app"
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
		shortID := app.GenerateShortKey(store, string(origURL))
		body := urlTemplate(shortID)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(body))
	}
}

func urlTemplate(id string) string {
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

func Router(store storage.Storager) chi.Router {
	r := chi.NewRouter()
	r.Post("/", CreateURL(store))  // POST /
	r.Get("/{key}", GetURL(store)) // GET /Fvdvgfgf
	return r
}
