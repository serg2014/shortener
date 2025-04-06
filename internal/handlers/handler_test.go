package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	type kv struct {
		key   string
		value string
	}
	tests := []struct {
		name    string
		want    want
		request *http.Request
		store   kv
	}{
		{
			name: "test #1",
			want: want{
				statusCode:  http.StatusBadRequest,
				response:    "bad id\n",
				contentType: "text/plain; charset=utf-8",
			},
			request: httptest.NewRequest(http.MethodGet, "/abcdefgh", nil),
			store:   kv{},
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
			store:   kv{"abcdefgh", "http://some.ru/123"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// создаём новый Recorder
			w := httptest.NewRecorder()
			if test.store.key != "" {
				store.Set(test.store.key, test.store.value)
			}
			GetURL(w, test.request)

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

func TestCreateURL(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		location    string
		response    string
	}
	type kv struct {
		key   string
		value string
	}
	tests := []struct {
		name    string
		want    want
		request *http.Request
		store   kv
	}{
		{
			name: "test #1",
			want: want{
				statusCode:  http.StatusCreated,
				response:    urlTemplate(""),
				contentType: "text/plain",
			},
			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader("http://some.ru/123")),
			store:   kv{"aaaaaa", "http://some.ru/123"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// создаём новый Recorder
			w := httptest.NewRecorder()
			if test.store.key != "" {
				store.Set(test.store.key, test.store.value)
			}
			CreateURL(w, test.request)

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
				_, ok := store.Get(id)
				assert.True(t, ok)
			}
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
