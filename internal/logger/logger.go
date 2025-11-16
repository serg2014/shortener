// Package logger add to logger http status of response and response size, request method, request uri...
package logger

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/serg2014/shortener/internal/auth"
)

// Log будет доступен всему коду как синглтон.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

// Write implement Write of http.ResponseWriter
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

// WriteHeader implement WriteHeader of http.ResponseWriter
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

// WithLogging добавляет дополнительный код для регистрации сведений о запросе
// и возвращает новый http.Handler.
func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		// функция Now() возвращает текущее время
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := &loggingResponseWriter{
			ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
			responseData:   responseData,
		}

		// точка, где выполняется хендлер pingHandler
		h.ServeHTTP(lw, r) // обслуживание оригинального запроса

		// Since возвращает разницу во времени между start
		// и моментом вызова Since. Таким образом можно посчитать
		// время выполнения запроса.
		duration := time.Since(start)

		// отправляем сведения о запросе в zap
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			userID = ""
		}
		Log.Info(
			"got incoming HTTP request",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", duration),
			zap.Int("status", responseData.status),
			zap.Int("size", responseData.size),
			zap.String("userID", string(userID)),
			zap.String("IP", r.Header.Get("X-Real-IP")),
		)
	}
	// возвращаем функционально расширенный хендлер
	return http.HandlerFunc(logFn)
}

// loggerInterceptor
func LoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// выполняем действия перед вызовом метода
	start := time.Now()
	// Возвращаем ответ и ошибку от фактического обработчика
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	// отправляем сведения о запросе в zap
	userID, errUser := auth.GetUserID(ctx)
	if errUser != nil {
		userID = ""
	}

	status := status.Convert(err)
	Log.Info(
		"got gRPC request",
		zap.String("method", info.FullMethod),
		zap.Duration("duration", duration),
		zap.Int("status", int(status.Code())),
		// TODO is it really need?
		// zap.Int("size", responseData.size),
		zap.String("userID", string(userID)),
		zap.String("IP", GetIP(ctx)),
	)
	return resp, err
}

// GetIP
func GetIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("X-Real-IP")
		if len(values) > 0 {
			// ключ содержит слайс строк, получаем первую строку
			return values[0]
		}
	}
	return ""
}
