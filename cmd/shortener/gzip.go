package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/serg2014/shortener/internal/logger"
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	compression bool
}

func newCompressWriter(gzw *gzip.Writer, w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:           w,
		zw:          gzw,
		compression: true,
	}
}

// Header implement Header from http.ResponseWriter
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write implement Write from http.ResponseWriter
func (c *compressWriter) Write(p []byte) (int, error) {
	if c.compression {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

// WriteHeader implement WriteHeader from http.ResponseWriter
func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	} else {
		c.compression = false
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	if c.compression {
		return c.zw.Close()
	}
	return nil
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read implemet Read from io.ReadCloser
func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close implemet Close from io.ReadCloser
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// gzipMiddleware middleware для сжатия/разжатия gzip
func gzipMiddleware(pool *sync.Pool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
			// который будем передавать следующей функции
			ow := w

			// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
			acceptEncoding := strings.Split(r.Header.Get("Accept-Encoding"), ", ")
			supportsGzip := slices.Index(acceptEncoding, "gzip") != -1
			logger.Log.Sugar().Infof("acceptEncoding: %s, supportsGzip: %t", acceptEncoding, supportsGzip)
			if supportsGzip {
				gzw := pool.Get().(*gzip.Writer)
				gzw.Reset(w)
				// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
				cw := newCompressWriter(gzw, w)
				// меняем оригинальный http.ResponseWriter на новый
				ow = cw

				defer func() {
					// не забываем отправить клиенту все сжатые данные после завершения middleware
					cw.Close()
					// вернуть в буфер
					pool.Put(gzw)
				}()
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
}
