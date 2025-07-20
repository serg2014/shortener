package auth

import (
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

var ErrCookieUserID = fmt.Errorf("no valid cookie %s", CookieName)
var ErrSetCookieUserID = fmt.Errorf("no set-cookie %s", CookieName)

var SECRET = []byte("somesecret")

func sign(value, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(value)
	return base64.RawStdEncoding.EncodeToString(h.Sum(nil))
}

func generateUserID() string {
	time := strconv.FormatInt(time.Now().Unix(), 10)
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawStdEncoding.EncodeToString([]byte(time)) + base64.RawStdEncoding.EncodeToString(b)
}

func createToken(value string) string {
	signature := sign([]byte(value), SECRET)
	return fmt.Sprintf("%s%s%s", value, TokenSep, signature)
}

func parseToken(token string) ([]string, error) {
	items := strings.Split(token, TokenSep)
	if len(items) != 2 {
		return nil, errors.New("bad token")
	}
	return items, nil
}

func isValidToken(token string) bool {
	items, err := parseToken(token)
	if err != nil {
		return false
	}
	return sign([]byte(items[0]), SECRET) == items[1]
}

func setCookieUserID(w http.ResponseWriter, value string) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    createToken(value),
		Path:     "/",
		HttpOnly: true, // Доступ только через HTTP, защита от XSS
		// Secure:   true,                    // Только HTTPS
		SameSite: http.SameSiteStrictMode, // Защита от CSRF
	}
	http.SetCookie(w, cookie)
}

func GetUserIDFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || !isValidToken(cookie.Value) {
		return "", ErrCookieUserID
	}
	items, _ := parseToken(cookie.Value)
	return items[0], nil
}

func GetUserID(w http.ResponseWriter, r *http.Request) (string, error) {
	userID, err := GetUserIDFromCookie(r)
	if err == nil {
		return userID, nil
	}

	httpHeader := w.Header()
	values := httpHeader.Values("Set-Cookie")
	if len(values) != 0 {
		for _, val := range values {
			cookie, err := http.ParseSetCookie(val)
			if err != nil {
				continue
			}
			if cookie.Name == CookieName {
				userID = cookie.Value
				break
			}
		}
	}
	if userID == "" {
		return "", ErrSetCookieUserID
	}

	return userID, nil
}

func AuthMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := GetUserIDFromCookie(r)
		if err != nil {
			userID := generateUserID()
			setCookieUserID(w, userID)
		}
		/*
			cookie, err := r.Cookie(CookieName)
			if err != nil || !IsValidToken(cookie.Value) {
				setCookieUserID(w, generateUserID())
			}
		*/
		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	})
}
