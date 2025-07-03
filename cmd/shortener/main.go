package main

import (
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
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

func gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		logger.Log.Sugar().Infof("acceptEncoding: %s, supportsGzip: %s", acceptEncoding, supportsGzip)
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
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

func Router(store storage.Storager) chi.Router {
	r := chi.NewRouter()
	//r.Use(middleware.Logger)
	r.Use(logger.WithLogging)
	r.Use(gzipMiddleware)

	r.Post("/", handlers.CreateURL(store))  // POST /
	r.Get("/{key}", handlers.GetURL(store)) // GET /Fvdvgfgf
	r.Post("/api/shorten", handlers.CreateURL2(store))
	//r.Post("/", logger.RequestLogger(CreateURL(store)))  // POST /
	//r.Get("/{key}", logger.RequestLogger(GetURL(store))) // GET /Fvdvgfgf
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
	logger.Log.Info("Running server", zap.String("address", config.Config.String()))
	var store = storage.NewStorage(nil)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", config.Config.Host, config.Config.Port), Router(store))
}
