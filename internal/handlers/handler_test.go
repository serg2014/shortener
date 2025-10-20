package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/serg2014/shortener/internal/app"
	appmock "github.com/serg2014/shortener/internal/app/mock"
	"github.com/serg2014/shortener/internal/auth"
	"github.com/serg2014/shortener/internal/storage"
	"github.com/serg2014/shortener/internal/storage/mock"
)

func Test_noUser(t *testing.T) {
	w := httptest.NewRecorder()
	noUser(w, auth.ErrUserIDFromContext)
	resp := w.Result()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "no user\n", string(respBody))
}

func httptestNewRequest_test(method string, url string, body io.Reader, headers map[string]string, userID string) *http.Request {
	req := httptest.NewRequest(method, url, body)
	for k := range headers {
		req.Header.Add(k, headers[k])
	}
	if userID != "" {
		req = req.WithContext(
			auth.WithUser(req.Context(), (*auth.UserID)(&userID)),
		)
	}
	return req
}

type headers map[string]string
type want struct {
	statusCode int
	headers    headers
	body       string
}
type myTest struct {
	name      string
	a         *app.MyApp
	req       *http.Request
	storeMock []func() *gomock.Call
	genMock   []func() *gomock.Call
	want      want
}

func TestCreateURL(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища, а так же мок генерилки
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	gen := appmock.NewMockGenarator(ctrl)
	a := app.NewApp(store, gen)

	tests := []myTest{

		{
			name: "no user",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://original.url/123"),
				map[string]string{},
				"",
			),
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "no user\n",
			},
		},
		{
			name: "empty url",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(""),
				map[string]string{},
				"user1",
			),
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "empty url\n",
			},
		},
		{
			name: "problem with storage",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://ya.ru"),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(errors.New("some storage problem"))
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "problem with storage GetShort",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://ya.ru"),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(storage.ErrConflict)
				},
				func() *gomock.Call {
					return store.EXPECT().
						GetShort(gomock.Any(), "http://ya.ru").
						Return("", false, errors.New("some another storage problem"))
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "can not find origurl",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://ya.ru"),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(storage.ErrConflict)
				},
				func() *gomock.Call {
					return store.EXPECT().
						GetShort(gomock.Any(), "http://ya.ru").
						Return("", false, nil)
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "already created",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://ya.ru"),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(storage.ErrConflict)
				},
				func() *gomock.Call {
					return store.EXPECT().
						GetShort(gomock.Any(), "http://ya.ru").
						Return("a1234567", true, nil)
				},
			},
			want: want{
				statusCode: http.StatusConflict,
				headers: headers{
					"content-type":     "text/plain",
					"Content-Encoding": "",
				},
				body: "http://localhost:8080/a1234567",
			},
		},
		{
			name: "success created",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://ya.ru"),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(nil)
				},
			},
			want: want{
				statusCode: http.StatusCreated,
				headers: headers{
					"content-type":     "text/plain",
					"Content-Encoding": "",
				},
				body: "http://localhost:8080/a1234567",
			},
		},
	}

	runTests(t, tests, func(newa *app.MyApp) http.HandlerFunc { return CreateURL(newa) })
}

func runTests(t *testing.T, tests []myTest, f func(a *app.MyApp) http.HandlerFunc) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.genMock != nil {
				for i := range test.genMock {
					test.genMock[i]()
				}
			}
			if test.storeMock != nil {
				for i := range test.storeMock {
					test.storeMock[i]()
				}
			}
			handler := f(test.a)
			w := httptest.NewRecorder()
			handler(w, test.req)

			result := w.Result()
			assert.Equal(t, test.want.statusCode, result.StatusCode)

			for k := range test.want.headers {
				assert.Equal(t, test.want.headers[k], result.Header.Get(k), k)
			}

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			defer result.Body.Close()
			assert.Equal(t, test.want.body, string(body))
		})
	}
}

func TestCreateURLJson(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища, а так же мок генерилки
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	gen := appmock.NewMockGenarator(ctrl)
	a := app.NewApp(store, gen)

	tests := []myTest{
		{
			name: "error parse json. empty body",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("http://ya.ru"),
				map[string]string{},
				"user1",
			),
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "bad json\n",
			},
		},
		{
			name: "error parse json. wrong type",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("[1]"),
				map[string]string{},
				"user1",
			),
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "bad json\n",
			},
		},
		{
			name: "no user",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{}`),
				map[string]string{},
				"",
			),
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "no user\n",
			},
		},
		{
			name: "empty url",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader("{}"),
				map[string]string{},
				"user1",
			),
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "empty url\n",
			},
		},
		{
			name: "empty url2",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{"url":""}`),
				map[string]string{},
				"user1",
			),
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "empty url\n",
			},
		},
		//
		{
			name: "problem with storage",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{"url":"http://ya.ru"}`),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(errors.New("some storage problem"))
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "problem with storage GetShort",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{"url":"http://ya.ru"}`),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(storage.ErrConflict)
				},
				func() *gomock.Call {
					return store.EXPECT().
						GetShort(gomock.Any(), "http://ya.ru").
						Return("", false, errors.New("some another storage problem"))
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "can not find origurl",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{"url":"http://ya.ru"}`),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(storage.ErrConflict)
				},
				func() *gomock.Call {
					return store.EXPECT().
						GetShort(gomock.Any(), "http://ya.ru").
						Return("", false, nil)
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "already created",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{"url":"http://ya.ru"}`),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(storage.ErrConflict)
				},
				func() *gomock.Call {
					return store.EXPECT().
						GetShort(gomock.Any(), "http://ya.ru").
						Return("a1234567", true, nil)
				},
			},
			want: want{
				statusCode: http.StatusConflict,
				headers: headers{
					"content-type":     "application/json",
					"Content-Encoding": "",
				},
				body: `{"result":"http://localhost:8080/a1234567"}` + "\n",
			},
		},
		{
			name: "success created",
			a:    a,
			req: httptestNewRequest_test(
				http.MethodPost,
				"/",
				strings.NewReader(`{"url":"http://ya.ru"}`),
				map[string]string{},
				"user1",
			),
			genMock: []func() *gomock.Call{
				func() *gomock.Call {
					return gen.EXPECT().GenerateShortKey().Return("a1234567")
				},
			},
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Set(gomock.Any(), "a1234567", "http://ya.ru", "user1").
						Return(nil)
				},
			},
			want: want{
				statusCode: http.StatusCreated,
				headers: headers{
					"content-type":     "application/json",
					"Content-Encoding": "",
				},
				body: `{"result":"http://localhost:8080/a1234567"}` + "\n",
			},
		},
	}

	runTests(t, tests, func(newa *app.MyApp) http.HandlerFunc { return CreateURLJson(newa) })
}

func httptestNewRequest_test_chi(method string, url string, body io.Reader, headers map[string]string, userID string, chiKey string) *http.Request {
	r := httptestNewRequest_test(method, url, body, headers, userID)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("key", chiKey)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}
func TestGetURL(t *testing.T) {
	// создадим конроллер моков и экземпляр мок-хранилища, а так же мок генерилки
	ctrl := gomock.NewController(t)
	store := mock.NewMockStorager(ctrl)
	gen := appmock.NewMockGenarator(ctrl)
	a := app.NewApp(store, gen)

	tests := []myTest{
		{
			name: "bad url",
			a:    a,
			req: httptestNewRequest_test_chi(
				http.MethodGet,
				"/",
				strings.NewReader(""),
				map[string]string{},
				"user1",
				"",
			),
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Get(gomock.Any(), "").
						Return("", false, nil)
				},
			},
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "bad short url\n",
			},
		},
		{
			name: "deleted",
			a:    a,
			req: httptestNewRequest_test_chi(
				http.MethodGet,
				"/some_url",
				strings.NewReader(""),
				map[string]string{},
				"user1",
				"some_url",
			),
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Get(gomock.Any(), "some_url").
						Return("", false, storage.ErrDeleted)
				},
			},
			want: want{
				statusCode: http.StatusGone,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Gone\n",
			},
		},
		{
			name: "store error",
			a:    a,
			req: httptestNewRequest_test_chi(
				http.MethodGet,
				"/some_url",
				strings.NewReader(""),
				map[string]string{},
				"user1",
				"some_url",
			),
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Get(gomock.Any(), "some_url").
						Return("", false, errors.New("some store error"))
				},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "Internal Server Error\n",
			},
		},
		{
			name: "short url not found",
			a:    a,
			req: httptestNewRequest_test_chi(
				http.MethodGet,
				"/some_url",
				strings.NewReader(""),
				map[string]string{},
				"user1",
				"some_url",
			),
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Get(gomock.Any(), "some_url").
						Return("", false, nil)
				},
			},
			want: want{
				statusCode: http.StatusBadRequest,
				headers: headers{
					"content-type":     "text/plain; charset=utf-8",
					"Content-Encoding": "",
				},
				body: "bad short url\n",
			},
		},
		{
			name: "ok",
			a:    a,
			req: httptestNewRequest_test_chi(
				http.MethodGet,
				"/some_url",
				strings.NewReader(""),
				map[string]string{},
				"user1",
				"some_url",
			),
			storeMock: []func() *gomock.Call{
				func() *gomock.Call {
					return store.EXPECT().
						Get(gomock.Any(), "some_url").
						Return("http://ya.ru", true, nil)
				},
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				headers: headers{
					"content-type":     "text/plain",
					"Content-Encoding": "",
					"Location":         "http://ya.ru",
				},
				body: "",
			},
		},
	}

	runTests(t, tests, func(newa *app.MyApp) http.HandlerFunc { return GetURL(newa) })

}
