package main

import (
	"net/http"

	"github.com/serg2014/shortener/internal/config"
)

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
