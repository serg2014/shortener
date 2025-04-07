package main

import (
	"fmt"
	"net/http"

	"github.com/serg2014/shortener/internal/app"
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
	return http.ListenAndServe(fmt.Sprintf("%s:%d", app.Host, app.Port), handlers.Router())
}
