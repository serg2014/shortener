package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/handlers"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/storage"
)

const poolSize = 2

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// TODO добавить тесты
func gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := strings.Split(r.Header.Get("Accept-Encoding"), ", ")
		supportsGzip := slices.Index(acceptEncoding, "gzip") != -1
		logger.Log.Sugar().Infof("acceptEncoding: %s, supportsGzip: %t", acceptEncoding, supportsGzip)
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := strings.Split(r.Header.Get("Content-Encoding"), ", ")
		sendsGzip := slices.Index(contentEncoding, "gzip") != -1
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(ow, r)
	})
}

func Router(a *app.MyApp) chi.Router {
	r := chi.NewRouter()
	//r.Use(middleware.Logger)
	r.Use(logger.WithLogging)
	r.Use(auth.AuthMiddleware)
	r.Use(gzipMiddleware)

	r.Post("/", handlers.CreateURL(a))  // POST /
	r.Get("/{key}", handlers.GetURL(a)) // GET /Fvdvgfgf
	r.Post("/api/shorten", handlers.CreateURLJson(a))
	//r.Post("/", logger.RequestLogger(CreateURL(store)))  // POST /
	//r.Get("/{key}", logger.RequestLogger(GetURL(store))) // GET /Fvdvgfgf
	r.Get("/ping", handlers.Ping(a))
	r.Post("/api/shorten/batch", handlers.CreateURLBatch(a))
	r.Get("/api/user/urls", handlers.GetUserURLS(a))
	r.Delete("/api/user/urls", handlers.DeleteUserURLS(a))
	return r
}

func run() error {
	err := config.Config.InitConfig()
	if err != nil {
		return err
	}

	if err := logger.Initialize(config.Config.LogLevel); err != nil {
		return err
	}

	ctx := context.Background()
	store, err := storage.NewStorage(ctx, config.Config.FileStoragePath, config.Config.DatabaseDSN)
	if err != nil {
		return err
	}
	defer store.Close()

	app := app.NewApp(store)
	for i := 0; i < poolSize; i++ {
		go app.DeleteUserURLSBackground(ctx)
	}

	logger.Log.Info("Running server", zap.String("address", config.Config.String()), zap.String("storage", fmt.Sprintf("%T", store)))
	return http.ListenAndServe(fmt.Sprintf("%s:%d", config.Config.Host, config.Config.Port), Router(app))
}
