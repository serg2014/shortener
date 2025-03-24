// пакеты исполняемых приложений должны называться main
package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
)

const (
	port = 8080
	host = "localhost"
)

var short2orig = make(map[string]string, 10)

// функция main вызывается автоматически при запуске приложения
func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webhook)
	return http.ListenAndServe(fmt.Sprintf(":%v", port), mux)
}

// функция webhook — обработчик HTTP-запроса
func webhook(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" && r.Method == http.MethodPost {
		createURL(w, r)
	} else if r.Method == http.MethodGet {
		getURL(w, r)
	} else {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func createURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	origURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	shortURL := generateShortKey()
	short2orig[shortURL] = string(origURL)
	body := fmt.Sprintf("http://%s:%d/%s", host, port, shortURL)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(body))
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

func getURL(w http.ResponseWriter, r *http.Request) {
	origURL, err := getOrigURL(strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
}

func getOrigURL(id string) (string, error) {
	if origURL, ok := short2orig[id]; ok {
		return origURL, nil
	}
	return "", errors.New("bad id")
}
