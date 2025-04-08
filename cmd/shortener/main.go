package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

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

	flag.Var(app.NewConfig, "a", "Net address host:port")
	flag.StringVar(&app.NewURL, "b", "", "Like http://ya.ru")
	flag.Parse()

	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		err := app.NewConfig.Set(envRunAddr)
		if err != nil {
			return err
		}
	}

	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		app.NewURL = baseURL
	}

	if app.NewURL != "" {
		if !strings.HasSuffix(app.NewURL, "/") {
			app.NewURL = app.NewURL + "/"
		}
	}
	return http.ListenAndServe(fmt.Sprintf("%s:%d", app.NewConfig.Host, app.NewConfig.Port), handlers.Router())
}
