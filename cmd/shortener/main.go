package main

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/handlers"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/storage"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	err := config.Config.InitConfig()
	if err != nil {
		return err
	}

	if err := logger.Initialize(config.Config.LogLevel); err != nil {
		return err
	}
	logger.Log.Info("Running server", zap.String("address", config.Config.String()))
	var store = storage.NewStorage(nil)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", config.Config.Host, config.Config.Port), handlers.Router(store))
}
