package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CookieName = "user_id"
const TokenSep = "."

type UserID string

var ErrCookieUserID = fmt.Errorf("no valid cookie %s", CookieName)
var ErrUserIDFromContext = fmt.Errorf("no userid in context")
var ErrBadToken = errors.New("bad token")
var ErrBadSignature = errors.New("bad signature")

var secret = []byte("somesecret")

func sign(value, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(value)
	return base64.RawStdEncoding.EncodeToString(h.Sum(nil))
}

func generateUserID() UserID {
	time := strconv.FormatInt(time.Now().Unix(), 10)
	b := make([]byte, 16)
	rand.Read(b)
	return UserID(base64.RawStdEncoding.EncodeToString([]byte(time)) + base64.RawStdEncoding.EncodeToString(b))
}

func createToken(value UserID) string {
	signature := sign([]byte(value), secret)
	return fmt.Sprintf("%s%s%s", value, TokenSep, signature)
}

func checkToken(token string) (UserID, error) {
	items := strings.Split(token, TokenSep)
	if len(items) != 2 {
		return "", ErrBadToken
	}
	if sign([]byte(items[0]), secret) != items[1] {
		return "", ErrBadSignature
	}
	return UserID(items[0]), nil
}

func setCookieUserID(w http.ResponseWriter, value UserID) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    createToken(value),
		Path:     "/",
		HttpOnly: true,                    // Доступ только через HTTP, защита от XSS
		SameSite: http.SameSiteStrictMode, // Защита от CSRF
	}
	http.SetCookie(w, cookie)
}

func GetUserIDFromCookie(r *http.Request) (UserID, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "", ErrCookieUserID
	}
	userID, err := checkToken(cookie.Value)
	if err != nil {
		return "", err
	}
	return userID, nil
}

type userCtxKeyType string

const userCtxKey userCtxKeyType = "userID"

func WithUser(ctx context.Context, userID *UserID) context.Context {
	return context.WithValue(ctx, userCtxKey, userID)
}

// TODO ptr
func GetUserID(ctx context.Context) (UserID, error) {
	userID, ok := ctx.Value(userCtxKey).(*UserID)
	if !ok {
		return "", ErrUserIDFromContext
	}
	return *userID, nil
}

func AuthMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromCookie(r)
		if err != nil {
			userID = generateUserID()
			setCookieUserID(w, userID)
		}
		// сохраним в контекст
		ctx := WithUser(r.Context(), &userID)
		// TODO может надо Clone
		r2 := r.WithContext(ctx)
		*r = *r2

		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	})
}
