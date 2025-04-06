package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//func TestcreateURL(t *testing.T) {
//}

func TestGetURL(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		location    string
		response    string
	}
	tests := []struct {
		name    string
		want    want
		request *http.Request
		store   *storage.Storage
	}{
		{
			name: "test #1",
			want: want{
				statusCode:  http.StatusBadRequest,
				response:    "bad id\n",
				contentType: "text/plain; charset=utf-8",
			},
			request: httptest.NewRequest(http.MethodGet, "/abcdefgh", nil),
			store:   storage.NewStorage(nil),
		},
		{
			name: "test #2",
			want: want{
				statusCode:  http.StatusTemporaryRedirect,
				location:    "http://some.ru/123",
				response:    "",
				contentType: "text/plain",
			},
			request: httptest.NewRequest(http.MethodGet, "/abcdefgh", nil),
			store:   storage.NewStorage(map[string]string{"abcdefgh": "http://some.ru/123"}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// создаём новый Recorder
			w := httptest.NewRecorder()
			getURL(w, test.request, test.store)

			// получаем ответ
			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if test.want.location != "" {
				assert.Equal(t, test.want.location, res.Header.Get("Location"))
			}
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, test.want.response, string(resBody))
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func Test_createURL(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		location    string
		response    string
	}
	tests := []struct {
		name    string
		want    want
		request *http.Request
		store   *storage.Storage
	}{
		{
			name: "test #1",
			want: want{
				statusCode:  http.StatusCreated,
				response:    urlTemplate(app.Host, app.Port, ""),
				contentType: "text/plain",
			},
			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader("http://some.ru/123")),
			store:   storage.NewStorage(map[string]string{"aaaaaa": "http://some.ru/123"}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// создаём новый Recorder
			w := httptest.NewRecorder()
			createURL(w, test.request, test.store)

			// получаем ответ
			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if test.want.location != "" {
				assert.Equal(t, test.want.location, res.Header.Get("Location"))
			}
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			id, ok := strings.CutPrefix(string(resBody), test.want.response)
			if assert.True(t, ok) {
				_, ok := test.store.Get(id)
				assert.True(t, ok)
			}
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
