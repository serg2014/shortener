package main

import (
	"context"
	"net/http"

	"github.com/serg2014/shortener/internal/config"
	"github.com/serg2014/shortener/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TrustedNetsMiddleware middleware for checking request from trusted net
func TrustedNetsMiddleware(trustedNet config.TrustedSubnet) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !trustedNet.IsTrusted(r.Header.Get("X-Real-IP")) {
				code := http.StatusForbidden
				http.Error(w, http.StatusText(code), code)
				return
			}
			// передаём управление хендлеру
			h.ServeHTTP(w, r)
		})
	}
}

// trustedInterceptor
func trustedInterceptor(trustedNet config.TrustedSubnet) func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// выполняем действия перед вызовом метода
		if info.FullMethod == "/shortener.ShortenerService/InternalStats" {
			// TODO getIP define in logging. is it true way?
			trust := trustedNet.IsTrusted(logger.GetIP(ctx))
			if !trust {
				code := codes.PermissionDenied
				return nil, status.Error(code, code.String())
			}
		}
		// Возвращаем ответ и ошибку от фактического обработчика
		return handler(ctx, req)
	}
}
