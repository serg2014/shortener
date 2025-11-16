// Package auth
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// CookieName name of cookie for saving userid
const CookieName = "user_id"

// MetaUserName name of meta for saving userid
const MetaUserName = "User-ID"

// TokenSep seperator for cookie value
const TokenSep = "."

// UserID type
type UserID string

// ErrCookieUserID error for empty cookie CookieName
var ErrCookieUserID = fmt.Errorf("no valid cookie %s", CookieName)

// ErrUserIDFromContext error when no userid in context
var ErrUserIDFromContext = fmt.Errorf("no userid in context")

// ErrBadToken error for bad cookir value format
var ErrBadToken = errors.New("bad token")

// ErrBadSignature error for signature
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

// GetUserIDFromCookie get userid from cookie user_id
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

// GetUserIDFromMeta get userid from meta grpc
func GetUserIDFromMeta(ctx context.Context) (UserID, error) {
	value := ""
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get(MetaUserName)
		if len(values) > 0 {
			// ключ содержит слайс строк, получаем первую строку
			value = values[0]
		}
	}
	if value == "" {
		return "", ErrCookieUserID
	}
	userID, err := checkToken(value)
	if err != nil {
		return "", err
	}
	return userID, nil
}

type userCtxKeyType string

const userCtxKey userCtxKeyType = "userID"

// WithUser helper set userid in context
func WithUser(ctx context.Context, userID *UserID) context.Context {
	return context.WithValue(ctx, userCtxKey, userID)
}

// GetUserID get userid from context
// TODO ptr
func GetUserID(ctx context.Context) (UserID, error) {
	userID, ok := ctx.Value(userCtxKey).(*UserID)
	if !ok {
		return "", ErrUserIDFromContext
	}
	return *userID, nil
}

// AuthMiddleware get userid from cookie and save it in context.
// Or create userid, save it into context and set cookie
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

// AuthInterceptor get userid from meta grpc and save it in context.
// Or create userid, save it into context and meta
func AuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// выполняем действия перед вызовом метода
	if info.FullMethod != "/shortener.ShortenerService/InternalStats" {
		userID, err := GetUserIDFromMeta(ctx)
		if err != nil {
			userID = generateUserID()
			ctx = metadata.AppendToOutgoingContext(ctx, MetaUserName, createToken(userID))
		}

		// сохраним в контекст
		ctx = WithUser(ctx, &userID)
	}

	// Возвращаем ответ и ошибку от фактического обработчика
	return handler(ctx, req)
}
