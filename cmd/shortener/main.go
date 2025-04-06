package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/serg2014/shortener/internal/handlers"
)

// функция main вызывается автоматически при запуске приложения
func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {
	r := chi.NewRouter()
	r.Post("/", handlers.CreateURL) // POST /
	r.Get("/", handlers.GetURL)     // GET /Fvdvgfgf
	return http.ListenAndServe(":8080", r)
}
