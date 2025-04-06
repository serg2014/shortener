package handlers

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/storage"
)

var store = storage.NewStorage(nil)

// функция webhook — обработчик HTTP-запроса
func Webhook(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" && r.Method == http.MethodPost {
		createURL(w, r, store)
	} else if r.Method == http.MethodGet {
		getURL(w, r, store)
	} else {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func createURL(w http.ResponseWriter, r *http.Request, store *storage.Storage) {
	w.Header().Set("Content-Type", "text/plain")
	origURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortURL := generateShortKey()
	store.Set(shortURL, string(origURL))
	body := urlTemplate(app.Host, app.Port, shortURL)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(body))
}

func urlTemplate(host string, port int, id string) string {
	return fmt.Sprintf("http://%s:%d/%s", host, port, id)
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

func getURL(w http.ResponseWriter, r *http.Request, store *storage.Storage) {
	origURL, err := getOrigURL(store, strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
}

func getOrigURL(store *storage.Storage, id string) (string, error) {
	if origURL, ok := store.Get(id); ok {
		return origURL, nil
	}
	return "", errors.New("bad id")
}
