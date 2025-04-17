package main

import (
	"fmt"
	"net/http"

	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/handlers"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	err := config.NewConfig.InitConfig()
	if err != nil {
		return err
	}

	return http.ListenAndServe(fmt.Sprintf("%s:%d", config.NewConfig.Host, config.NewConfig.Port), handlers.Router())
}
