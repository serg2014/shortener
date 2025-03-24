// пакеты исполняемых приложений должны называться main
package main

import (
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
		createUrl(w, r)
	} else if r.Method == http.MethodGet {
		getUrl(w, r)
	} else {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func createUrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	origUrl, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
	}
	shortUrl := generateShortKey()
	short2orig[shortUrl] = string(origUrl)
	body := fmt.Sprintf("http://%s:%d/%s", host, port, shortUrl)
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

func getUrl(w http.ResponseWriter, r *http.Request) {
	origUrl := getOrigUrl(strings.TrimPrefix(r.URL.Path, "/"))
	w.Header().Set("Content-Type", "text/plain")
	http.Redirect(w, r, origUrl, http.StatusTemporaryRedirect)
}

func getOrigUrl(id string) string {
	if origUrl, ok := short2orig[id]; ok {
		return origUrl
	}
	return ""
}
