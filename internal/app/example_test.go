package app_test

import (
	"context"

	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/storage"
)

func Example() {
	ctx := context.Background()
	store, err := storage.NewStorage(ctx, config.Config.FileStoragePath, config.Config.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	app := app.NewApp(store, nil)
	app.Ping(ctx)
}
