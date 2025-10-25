package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createToken(t *testing.T) {
	tests := []struct {
		name   string
		userID UserID
		expect string
	}{
		{
			name:   "createToken",
			userID: "some_user_id",
			expect: "some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := createToken(test.userID)
			assert.Equal(t, test.expect, got)
		})
	}
}

func Test_checkToken(t *testing.T) {
	type want struct {
		err    error
		userID UserID
	}
	tests := []struct {
		name   string
		token  string
		expect want
	}{
		{
			name:  "good token",
			token: "some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ",
			expect: want{
				err:    nil,
				userID: UserID("some_user_id"),
			},
		},
		{
			name:  "bad format",
			token: "some_user_id",
			expect: want{
				err:    ErrBadToken,
				userID: UserID(""),
			},
		},
		{
			name:  "bad format2",
			token: "some_user_id.some.some",
			expect: want{
				err:    ErrBadToken,
				userID: UserID(""),
			},
		},
		{
			name:  "bad signature",
			token: "some_user_id.some_signature",
			expect: want{
				err:    ErrBadSignature,
				userID: UserID(""),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userID, err := checkToken(test.token)
			if test.expect.err == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expect.userID, userID)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, test.expect.err)
		})
	}
}

func TestGetUserIDFromCookie(t *testing.T) {
	type want struct {
		userID UserID
		err    error
	}
	tests := []struct {
		name      string
		cookieVal string
		expect    want
	}{
		{
			name:      "good cookie",
			cookieVal: "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ",
			expect: want{
				userID: UserID("some_user_id"),
				err:    nil,
			},
		},
		{
			name:      "no cookie",
			cookieVal: "another_cookie=1",
			expect: want{
				userID: UserID(""),
				err:    ErrCookieUserID,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				"GET",
				"http://localhost/",
				nil,
			)
			req.Header.Add("cookie", test.cookieVal)

			userID, err := GetUserIDFromCookie(req)
			if test.expect.err == nil {
				require.NoError(t, err)
				assert.Equal(t, userID, test.expect.userID)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, test.expect.err)
		})
	}

}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		cookieVal   string
		nextHandler http.Handler
	}{
		{
			name:      "good userid from cookie",
			cookieVal: "user_id=some_user_id.kJusbumVnkwQSAX+zsXQscI83JIE1VVQcfrDpbXB7FQ",
			// create a handler to use as "next" which will verify the request
			nextHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				val := r.Context().Value(userCtxKey)
				require.NotNil(t, val, fmt.Sprintf("%s not present in context", userCtxKey))
				userID, ok := val.(*UserID)
				require.Equal(t, true, ok)
				assert.Equal(t, UserID("some_user_id"), *userID)
			}),
		},
		{
			name:      "bad cookie. generate new userid",
			cookieVal: "user_id=some_user_id",
			// create a handler to use as "next" which will verify the request
			nextHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				val := r.Context().Value(userCtxKey)
				require.NotNil(t, val, fmt.Sprintf("%s not present in context", userCtxKey))
				userID, ok := val.(*UserID)
				require.Equal(t, true, ok)
				assert.Equal(t, len(*userID), 36)
			}),
		},
		{
			name:      "no cookie. generate new userid",
			cookieVal: "some_val=some_user_id",
			// create a handler to use as "next" which will verify the request
			nextHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				val := r.Context().Value(userCtxKey)
				require.NotNil(t, val, fmt.Sprintf("%s not present in context", userCtxKey))
				userID, ok := val.(*UserID)
				require.Equal(t, true, ok)
				assert.Equal(t, len(*userID), 36)
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create the handler to test, using our custom "next" handler
			handlerToTest := AuthMiddleware(test.nextHandler)

			// create a mock request to use
			req := httptest.NewRequest("GET", "http://localhost/", nil)
			req.Header.Add("cookie", test.cookieVal)

			// call the handler using a mock response recorder (we'll not use that anyway)
			handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}
