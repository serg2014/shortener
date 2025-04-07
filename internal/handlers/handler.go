package handlers

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/storage"
)

var store = storage.NewStorage(nil)

// функция webhook — обработчик HTTP-запроса
func Webhook(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" && r.Method == http.MethodPost {
		CreateURL(w, r)
	} else if r.Method == http.MethodGet {
		GetURL(w, r)
	} else {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func CreateURL(w http.ResponseWriter, r *http.Request) {
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

func urlTemplate(id string) string {
	return fmt.Sprintf("%s%s", app.URL(), id)
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

func GetURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "key")
	origURL, err := getOrigURL(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
}

func getOrigURL(id string) (string, error) {
	if origURL, ok := store.Get(id); ok {
		return origURL, nil
	}
	return "", errors.New("bad id")
}

func Router() chi.Router {
	r := chi.NewRouter()
	r.Post("/", CreateURL)  // POST /
	r.Get("/{key}", GetURL) // GET /Fvdvgfgf
	return r
}
