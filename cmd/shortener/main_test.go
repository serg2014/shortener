package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/serg2014/shortener/internal/app"
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, req *http.Request) (*http.Response, string) {
	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestGetURL(t *testing.T) {
	store, err := storage.NewStorageMemory()
	require.NoError(t, err)

	a := app.NewApp(store)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

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
	type reqParam struct {
		method     string
		url        string
		body       io.Reader
		setHeaders map[string]string
	}

	tests := []struct {
		name     string
		want     want
		reqParam reqParam
		store    kv
	}{
		{
			name: "test 1",
			want: want{
				statusCode:  http.StatusBadRequest,
				response:    "bad short url\n",
				contentType: "text/plain; charset=utf-8",
			},
			reqParam: reqParam{http.MethodGet, "/abcdef12", nil, map[string]string{"Accept-Encoding": ""}},
			store:    kv{},
		},
		{
			name: "test 2",
			want: want{
				statusCode:  http.StatusTemporaryRedirect,
				location:    "http://some.ru/123",
				response:    "",
				contentType: "text/plain",
			},
			reqParam: reqParam{http.MethodGet, "/abcdefgh", nil, map[string]string{"Accept-Encoding": ""}},
			store:    kv{"abcdefgh", "http://some.ru/123"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.store.key != "" {
				userID := ""
				a.Set(t.Context(), test.store.key, test.store.value, auth.UserID(userID))
			}
			req, err := http.NewRequest(test.reqParam.method, ts.URL+test.reqParam.url, test.reqParam.body)
			require.NoError(t, err)
			if test.reqParam.setHeaders != nil {
				for k, v := range test.reqParam.setHeaders {
					req.Header.Set(k, v)
				}

			}
			//  на этот код полчил ошибку internal/handlers/handler_test.go:89:32: response body must be closed
			// resp, resBody := testRequest(t, ts, req)

			//resp, data := testRequest(t, ts, req)
			//====
			client := ts.Client()
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			//=====
			// проверяем код ответа
			assert.Equal(t, test.want.statusCode, resp.StatusCode)

			if test.want.location != "" {
				assert.Equal(t, test.want.location, resp.Header.Get("Location"))
			}

			assert.Equal(t, test.want.response, string(respBody))
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

func TestCreateURL(t *testing.T) {
	store, err := storage.NewStorageMemory()
	require.NoError(t, err)

	a := app.NewApp(store)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

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
	type reqParam struct {
		method string
		url    string
		body   io.Reader
	}
	tests := []struct {
		name     string
		want     want
		reqParam reqParam
		store    kv
	}{
		{
			name: "test #1",
			want: want{
				statusCode:  http.StatusCreated,
				response:    app.URLTemplate(""),
				contentType: "text/plain",
			},
			reqParam: reqParam{http.MethodPost, "/", strings.NewReader("http://some.ru/123")},
			store:    kv{"aaaaaa", "http://some.ru/123"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(test.reqParam.method, ts.URL+test.reqParam.url, test.reqParam.body)
			require.NoError(t, err)
			// получаем ответ
			//resp, data := testRequest(t, ts, req)
			//====
			client := ts.Client()
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			//=====
			// проверяем код ответа
			assert.Equal(t, test.want.statusCode, resp.StatusCode)

			id, ok := strings.CutPrefix(string(respBody), test.want.response)
			if assert.True(t, ok) {
				val, ok, err := a.Get(t.Context(), id)
				assert.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, test.store.value, val)
			}
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}
