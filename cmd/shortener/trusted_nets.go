package main

import (
	"net/http"
)

func TrustedNetsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	})
}
