package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/serg2014/shortener/internal/app"
	appmock "github.com/serg2014/shortener/internal/app/mock"
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/storage"
	"github.com/serg2014/shortener/internal/storage/mock"
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

	a := app.NewApp(store, nil)
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

	a := app.NewApp(store, nil)
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

type expect struct {
	code     int
	response string
	headers  map[string]string
}
type reqParam struct {
	method    string
	url       string
	body      io.Reader
	headers   map[string]string
	storeMock []func() *gomock.Call
	genMock   []func() *gomock.Call
}
type testsReqItem struct {
	name    string
	request reqParam
	expect  expect
}

func makeTestRequest(t *testing.T, ts *httptest.Server, client *http.Client, test testsReqItem) {
	if len(test.request.storeMock) != 0 {
		//gomock.InOrder(test.request.storeMock...)
		for _, f := range test.request.storeMock {
			gomock.InOrder(f())
		}

	}
	if len(test.request.genMock) != 0 {
		//gomock.InOrder(test.request.genMock...)
		for _, f := range test.request.genMock {
			gomock.InOrder(f())
		}
	}
	req, err := http.NewRequest(
		test.request.method,
		ts.URL+test.request.url,
		test.request.body,
	)
	require.NoError(t, err)
	if len(test.request.headers) != 0 {
		for k := range test.request.headers {
			req.Header.Add(k, test.request.headers[k])
		}
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
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

	assert.Equal(t, test.expect.code, resp.StatusCode)
	for k := range test.expect.headers {
		assert.Equal(t, test.expect.headers[k], resp.Header.Get(k), k)
	}

	respData := string(respBody)
	if resp.Header.Get("Content-Type") == "application/json" {
		assert.JSONEq(t, test.expect.response, respData)
		return
	}
	assert.Equal(t, test.expect.response, respData)
}

func Test_CreateURL_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	gen := appmock.NewMockGenarator(ctrl)
	a := app.NewApp(store, gen)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodPost,
				url:    "/",
				body:   strings.NewReader("http://original.url/123"),
				headers: map[string]string{
					"cookie": cookieVal,
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Set(gomock.Any(), gomock.Any(), "http://original.url/123", "some_user_id").
							Return(nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call {
						return gen.EXPECT().GenerateShortKey().Return("a1234567")
					},
				},
			},
			expect: expect{
				code:     http.StatusCreated,
				response: "http://localhost:8080/a1234567",
				headers: map[string]string{
					"Content-Type":     "text/plain",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok with gzip",
			request: reqParam{
				method: http.MethodPost,
				url:    "/",
				body:   strings.NewReader("http://original.url/123"),
				headers: map[string]string{
					"cookie":          cookieVal,
					"Accept-Encoding": "gzip",
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Set(gomock.Any(), gomock.Any(), "http://original.url/123", "some_user_id").
							Return(nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call {
						return gen.EXPECT().GenerateShortKey().Return("a1234567")
					},
				},
			},
			expect: expect{
				code:     http.StatusCreated,
				response: "http://localhost:8080/a1234567",
				headers: map[string]string{
					"Content-Type":     "text/plain",
					"Content-Encoding": "gzip",
				},
			},
		},
		{
			name: "empty body",
			request: reqParam{
				method: http.MethodPost,
				url:    "/",
				body:   strings.NewReader(""),
				headers: map[string]string{
					"cookie": cookieVal,
				},
				storeMock: []func() *gomock.Call{},
				genMock:   []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusBadRequest,
				response: "empty url\n",
				headers: map[string]string{
					"Content-Type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "empty body with gzip",
			request: reqParam{
				method: http.MethodPost,
				url:    "/",
				body:   strings.NewReader(""),
				headers: map[string]string{
					"cookie":          cookieVal,
					"Accept-Encoding": "gzip",
				},
				storeMock: []func() *gomock.Call{},
				genMock:   []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusBadRequest,
				response: "empty url\n",
				headers: map[string]string{
					"Content-Type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "conflict",
			request: reqParam{
				method: http.MethodPost,
				url:    "/",
				body:   strings.NewReader("http://original.url/123"),
				headers: map[string]string{
					"cookie": cookieVal,
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Set(gomock.Any(), gomock.Any(), "http://original.url/123", "some_user_id").
							Return(storage.ErrConflict)
					},
					func() *gomock.Call {
						return store.EXPECT().
							GetShort(gomock.Any(), "http://original.url/123").
							Return("a1234567", true, nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call {
						return gen.EXPECT().GenerateShortKey().Return("a1234567")
					},
				},
			},
			expect: expect{
				code:     http.StatusConflict,
				response: "http://localhost:8080/a1234567",
				headers: map[string]string{
					"Content-Type":     "text/plain",
					"Content-Encoding": "",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}
func TestGetURL_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	a := app.NewApp(store, nil)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodGet,
				url:    "/a1234567",
				headers: map[string]string{
					"cookie": cookieVal,
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Get(gomock.Any(), "a1234567").
							Return("http://some.long/url", true, nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusTemporaryRedirect,
				response: "",
				headers: map[string]string{
					"Content-Type":     "text/plain",
					"Location":         "http://some.long/url",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok with gzip",
			request: reqParam{
				method: http.MethodGet,
				url:    "/a1234567",
				headers: map[string]string{
					"cookie":          cookieVal,
					"accept-encoding": "gzip",
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Get(gomock.Any(), "a1234567").
							Return("http://some.long/url", true, nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusTemporaryRedirect,
				response: "",
				headers: map[string]string{
					"Content-Type":     "text/plain",
					"Location":         "http://some.long/url",
					"Content-Encoding": "",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}
func TestCreateURLJson_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	gen := appmock.NewMockGenarator(ctrl)
	a := app.NewApp(store, gen)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodPost,
				url:    "/api/shorten",
				headers: map[string]string{
					"cookie":       cookieVal,
					"Content-Type": "application/json",
				},
				body: strings.NewReader(`{"url":"http://original.url/123"}`),
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Set(gomock.Any(), "a1234567", "http://original.url/123", "some_user_id").
							Return(nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call {
						return gen.EXPECT().GenerateShortKey().Return("a1234567")
					},
				},
			},
			expect: expect{
				code:     http.StatusCreated,
				response: `{"result":"http://localhost:8080/a1234567"}`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok with gzip",
			request: reqParam{
				method: http.MethodPost,
				url:    "/api/shorten",
				headers: map[string]string{
					"cookie":          cookieVal,
					"Content-Type":    "application/json",
					"accept-encoding": "gzip",
				},
				body: strings.NewReader(`{"url":"http://original.url/123"}`),
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Set(gomock.Any(), "a1234567", "http://original.url/123", "some_user_id").
							Return(nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call {
						return gen.EXPECT().GenerateShortKey().Return("a1234567")
					},
				},
			},
			expect: expect{
				code:     http.StatusCreated,
				response: `{"result":"http://localhost:8080/a1234567"}`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "gzip",
				},
			},
		},
		{
			name: "conflict",
			request: reqParam{
				method: http.MethodPost,
				url:    "/api/shorten",
				headers: map[string]string{
					"cookie":          cookieVal,
					"Content-Type":    "application/json",
					"accept-encoding": "gzip",
				},
				body: strings.NewReader(`{"url":"http://original.url/123"}`),
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							Set(gomock.Any(), "a1234567", "http://original.url/123", "some_user_id").
							Return(storage.ErrConflict)
					},
					func() *gomock.Call {
						return store.EXPECT().
							GetShort(gomock.Any(), "http://original.url/123").
							Return("a1234567", true, nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call {
						return gen.EXPECT().GenerateShortKey().Return("a1234567")
					},
				},
			},
			expect: expect{
				code:     http.StatusConflict,
				response: `{"result":"http://localhost:8080/a1234567"}`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}

func TestPing_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	a := app.NewApp(store, nil)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodGet,
				url:    "/ping",
				headers: map[string]string{
					"cookie": cookieVal,
				},
				body: nil,
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().Ping(gomock.Any()).Return(nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusOK,
				response: "",
				headers: map[string]string{
					"Content-Type":     "",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok without cookie",
			request: reqParam{
				method:  http.MethodGet,
				url:     "/ping",
				headers: map[string]string{},
				body:    nil,
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().Ping(gomock.Any()).Return(nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusOK,
				response: "",
				headers: map[string]string{
					"Content-Type":     "",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok with gzip",
			request: reqParam{
				method: http.MethodGet,
				url:    "/ping",
				headers: map[string]string{
					"cookie":          cookieVal,
					"accept-encoding": "gzip",
				},
				body: nil,
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().Ping(gomock.Any()).Return(nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusOK,
				response: "",
				headers: map[string]string{
					"Content-Type":     "",
					"Content-Encoding": "gzip",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}

func TestCreateURLBatch_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	gen := appmock.NewMockGenarator(ctrl)
	a := app.NewApp(store, gen)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodPost,
				url:    "/api/shorten/batch",
				headers: map[string]string{
					"cookie":       cookieVal,
					"Content-Type": "application/json",
				},
				body: strings.NewReader(`[
				{"correlation_id":"one","original_url":"http://original.url/123"},
				{"correlation_id":"two","original_url":"http://original.url/124"}
				]`),
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							SetBatch(gomock.Any(), gomock.Any(), "some_user_id").
							Return(nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call { return gen.EXPECT().GenerateShortKey().Return("a1234567") },
					func() *gomock.Call { return gen.EXPECT().GenerateShortKey().Return("a1234568") },
				},
			},
			expect: expect{
				code: http.StatusCreated,
				response: `[
				{"correlation_id":"one", "short_url":"http://localhost:8080/a1234567"},
				{"correlation_id":"two", "short_url":"http://localhost:8080/a1234568"}
				]`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok with gzip",
			request: reqParam{
				method: http.MethodPost,
				url:    "/api/shorten/batch",
				headers: map[string]string{
					"cookie":          cookieVal,
					"Content-Type":    "application/json",
					"accept-encoding": "gzip",
				},
				body: strings.NewReader(`[
				{"correlation_id":"one","original_url":"http://original.url/123"},
				{"correlation_id":"two","original_url":"http://original.url/124"}
				]`),
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						return store.EXPECT().
							SetBatch(gomock.Any(), gomock.Any(), "some_user_id").
							Return(nil)
					},
				},
				genMock: []func() *gomock.Call{
					func() *gomock.Call { return gen.EXPECT().GenerateShortKey().Return("a1234567") },
					func() *gomock.Call { return gen.EXPECT().GenerateShortKey().Return("a1234568") },
				},
			},
			expect: expect{
				code: http.StatusCreated,
				response: `[
				{"correlation_id":"one", "short_url":"http://localhost:8080/a1234567"},
				{"correlation_id":"two", "short_url":"http://localhost:8080/a1234568"}
				]`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "gzip",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}

func TestGetUserURLS_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	a := app.NewApp(store, nil)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodGet,
				url:    "/api/user/urls",
				headers: map[string]string{
					"cookie": cookieVal,
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						data := []storage.Item{
							{
								ShortURL:    "a1234567",
								OriginalURL: "http://some.url/123",
							},
							{
								ShortURL:    "a1234568",
								OriginalURL: "http://some.url/1234",
							},
						}
						return store.EXPECT().
							GetUserURLS(gomock.Any(), "some_user_id").
							Return(data, nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code: http.StatusOK,
				response: `[
				{"original_url":"http://some.url/123", "short_url":"http://localhost:8080/a1234567"},
				{"original_url":"http://some.url/1234", "short_url":"http://localhost:8080/a1234568"}
				]`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "",
				},
			},
		},
		{
			name: "ok with gzip",
			request: reqParam{
				method: http.MethodGet,
				url:    "/api/user/urls",
				headers: map[string]string{
					"cookie":          cookieVal,
					"accept-encoding": "gzip",
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						data := []storage.Item{
							{
								ShortURL:    "a1234567",
								OriginalURL: "http://some.url/123",
							},
							{
								ShortURL:    "a1234568",
								OriginalURL: "http://some.url/1234",
							},
						}
						return store.EXPECT().
							GetUserURLS(gomock.Any(), "some_user_id").
							Return(data, nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code: http.StatusOK,
				response: `[
				{"original_url":"http://some.url/123", "short_url":"http://localhost:8080/a1234567"},
				{"original_url":"http://some.url/1234", "short_url":"http://localhost:8080/a1234568"}
				]`,
				headers: map[string]string{
					"Content-Type":     "application/json",
					"Content-Encoding": "gzip",
				},
			},
		},
		{
			name: "no data",
			request: reqParam{
				method: http.MethodGet,
				url:    "/api/user/urls",
				headers: map[string]string{
					"cookie": cookieVal,
				},
				storeMock: []func() *gomock.Call{
					func() *gomock.Call {
						data := []storage.Item{}
						return store.EXPECT().
							GetUserURLS(gomock.Any(), "some_user_id").
							Return(data, nil)
					},
				},
				genMock: []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusNoContent,
				response: ``,
				headers: map[string]string{
					"Content-Type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}

func TestDeleteUserURLS_database(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	a := app.NewApp(store, nil)
	ts := httptest.NewServer(Router(a))
	defer ts.Close()

	// отключить принудительное выставление content-encoding: gzip
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	// не ходить по редиректам
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	const cookieVal = "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ"
	tests := []testsReqItem{
		{
			name: "ok",
			request: reqParam{
				method: http.MethodDelete,
				url:    "/api/user/urls",
				headers: map[string]string{
					"cookie":       cookieVal,
					"content-type": "",
				},
				body:      strings.NewReader(`["6qxTVvsy", "RTfd56hn", "Jlfd67ds"]`),
				storeMock: []func() *gomock.Call{},
				genMock:   []func() *gomock.Call{},
			},
			expect: expect{
				code:     http.StatusAccepted,
				response: "",
				headers: map[string]string{
					"Content-Type":     "",
					"Content-Encoding": "",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeTestRequest(t, ts, client, test)
		})
	}
}
