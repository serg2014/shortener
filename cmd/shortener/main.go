package main

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/handlers"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/storage"
)

// TODO вынести в конфиг
// poolSize кол-во горутин обрабатывающих удаление урлов.
const poolSize = 10

// waitSecBeforeShutdown how many seconds wait before force shutdown
const waitSecBeforeShutdown = 5 * time.Second

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func valueOrNA(str string) string {
	if str == "" {
		return "N/A"
	}
	return str
}

func main() {
	fmt.Printf("Build version: %s\n", valueOrNA(buildVersion))
	fmt.Printf("Build date: %s\n", valueOrNA(buildDate))
	fmt.Printf("Build commit: %s\n", valueOrNA(buildCommit))
	if err := run(); err != nil {
		panic(err)
	}
}

// Router set up routes and return chi.Router
func Router(a *app.MyApp) chi.Router {
	r := chi.NewRouter()
	r.Route("/debug", func(r chi.Router) {
		// add pprof
		r.Mount("/", middleware.Profiler())
	})

	pool := &sync.Pool{
		New: func() any {
			return gzip.NewWriter(nil)
		},
	}

	r.Route("/", func(r chi.Router) {
		r.Use(logger.WithLogging)
		r.Use(gzipMiddleware(pool))

		r.Group(func(r chi.Router) {
			r.Use(TrustedNetsMiddleware)
			r.Get("/api/internal/stats", handlers.InternalStats(a))

		})
		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware)

			r.Post("/", handlers.CreateURL(a))
			r.Get("/{key}", handlers.GetURL(a))
			r.Post("/api/shorten", handlers.CreateURLJson(a))
			r.Get("/ping", handlers.Ping(a))
			r.Post("/api/shorten/batch", handlers.CreateURLBatch(a))
			r.Get("/api/user/urls", handlers.GetUserURLS(a))
			r.Delete("/api/user/urls", handlers.DeleteUserURLS(a))
		})
	})
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

	app := app.NewApp(store, nil)

	srv := http.Server{
		Addr:    config.Config.ServerAddress.String(),
		Handler: Router(app),
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			app.DeleteUserURLsBackground(ctx)
			logger.Log.Info("Stop delete gorutine")
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		// создаем контекст, который будет отменен при получении сигнала
		ctxS, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		defer stop()

		select {
		// 	ждем сигнала от ОС
		case <-ctxS.Done():
			logger.Log.Info("catch signal")
		// ждем отмены контекста
		case <-ctx.Done():
			logger.Log.Info("stop")
		}

		ctxT, cancelT := context.WithTimeout(context.Background(), waitSecBeforeShutdown)
		defer cancelT()
		if err := srv.Shutdown(ctxT); err != nil {
			logger.Log.Info("Server forced to shutdown", zap.Error(err))
		}
	}()

	logger.Log.Info("Try running server",
		zap.String("address", config.Config.ServerAddress.String()),
		zap.String("storage", fmt.Sprintf("%T", store)),
		zap.Bool("https", config.Config.HTTPS),
	)
	err = ListenAndServe(&srv, config.Config.HTTPS)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("ListenAndServe: %w", err)
	}

	// отменяем контекст, чтобы завершить горутины
	cancel()

	wg.Wait()
	logger.Log.Info("Server is shutdown")
	return nil
}

// ListenAndServe - srv.ListenAndServe or srv.ListenAndServeTLS
func ListenAndServe(srv *http.Server, isHTTPS bool) error {
	// http
	if !isHTTPS {
		return srv.ListenAndServe()
	}
	// https
	cert, pk, callback, err := getCertPK()
	if err != nil {
		return err
	}
	defer callback()
	logger.Log.Info("Paths", zap.String("cert path", cert), zap.String("pk path", pk))

	return srv.ListenAndServeTLS(cert, pk)
}
