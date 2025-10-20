package main

import (
	"compress/gzip"
	"io"
	"net/http"
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
