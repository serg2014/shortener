package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware_compression(t *testing.T) {
	data := []byte("Hello GO world!")
	tests := []struct {
		headers     map[string]string
		nextHandler http.Handler
		name        string
	}{
		{
			name: "ok compression",
			headers: map[string]string{
				"accept-encoding": "gzip",
			},
			// create a handler to use as "next" which will verify the request
			nextHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write(data)
			}),
		},
	}

	pool := &sync.Pool{
		New: func() any {
			var buf bytes.Buffer
			return gzip.NewWriter(&buf)
		},
	}
	gzm := gzipMiddleware(pool)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create the handler to test, using our custom "next" handler
			handlerToTest := gzm(test.nextHandler)

			// create a mock request to use
			req := httptest.NewRequest("GET", "http://localhost/", nil)
			if len(test.headers) != 0 {
				for k := range test.headers {
					req.Header.Add(k, test.headers[k])
				}
			}

			// call the handler using a mock response recorder (we'll not use that anyway)
			w := httptest.NewRecorder()
			handlerToTest.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			respReader := resp.Body
			if resp.Header.Get("Content-Encoding") == "gzip" {
				gzipReader, err := gzip.NewReader(resp.Body)
				require.NoError(t, err, "Error creating gzip reader")
				defer gzipReader.Close()
				respReader = gzipReader
			}
			respBody, err := io.ReadAll(respReader)
			require.NoError(t, err)
			assert.Equal(t, data, respBody)
		})
	}
}

func TestGzipMiddleware_decompression(t *testing.T) {
	data := []byte("Hello GO world!")
	tests := []struct {
		name        string
		nextHandler http.Handler
		headers     map[string]string
		data        []byte
	}{
		{
			name: "ok decompression",
			data: data,
			headers: map[string]string{
				"Content-Encoding": "gzip",
			},
			// create a handler to use as "next" which will verify the request
			nextHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				readData, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, data, readData)
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	pool := &sync.Pool{
		New: func() any {
			var buf bytes.Buffer
			return gzip.NewWriter(&buf)
		},
	}
	gzm := gzipMiddleware(pool)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create the handler to test, using our custom "next" handler
			handlerToTest := gzm(test.nextHandler)

			// create a mock request to use
			var buf bytes.Buffer
			gzw := gzip.NewWriter(&buf)
			_, err := gzw.Write(test.data)
			require.NoError(t, err)
			err = gzw.Close()
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "http://localhost/", &buf)
			if len(test.headers) != 0 {
				for k := range test.headers {
					req.Header.Add(k, test.headers[k])
				}
			}

			// call the handler using a mock response recorder (we'll not use that anyway)
			w := httptest.NewRecorder()
			handlerToTest.ServeHTTP(w, req)
		})
	}
}
