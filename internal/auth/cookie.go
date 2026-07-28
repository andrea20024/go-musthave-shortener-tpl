package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

const CookieName = "userID"

type contextKey string

const UserIDContextKey contextKey = "userID"

var key []byte

func Init(keyStr string) {
	key = []byte(keyStr)
}

func CookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := getOrCreateUser(r, w)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserIDContextKey, userID)))
	})
}

func getOrCreateUser(r *http.Request, w http.ResponseWriter) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil || !verify(cookie.Value) {
		userID := newID()
		setCookie(w, userID)
		return userID
	}
	parts := strings.SplitN(cookie.Value, "|", 2)
	return parts[0]
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func setCookie(w http.ResponseWriter, userID string) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: sign(userID), Path: "/", HttpOnly: true})
}

func sign(v string) string { return v + "|" + mac(v) }

func mac(v string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(v))
	return hex.EncodeToString(h.Sum(nil))
}

func verify(s string) bool {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) != 2 {
		return false
	}
	return hmac.Equal([]byte(parts[1]), []byte(mac(parts[0])))
}
