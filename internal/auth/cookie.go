// Package auth provides cookie-based user identification for the URL
// shortener service.
//
// Each incoming request is assigned a unique user ID stored in an HMAC-signed
// cookie. The user ID is propagated into the request context via
// UserIDContextKey and can be retrieved by handlers to associate shortened URLs
// with specific users.
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

// CookieName is the name of the cookie used to store the user identifier.
const CookieName = "userID"

// contextKey is an unexported type used for context keys defined in this package.
// This prevents collisions between keys defined in multiple packages.
type contextKey string

// UserIDContextKey is the context key under which the authenticated user ID
// is stored in the request context.
const UserIDContextKey contextKey = "userID"

var key []byte

// Init initializes the signing key used to verify and create user cookies.
// This must be called during application startup with a sufficiently long,
// cryptographically random secret.
func Init(keyStr string) {
	key = []byte(keyStr)
}

// CookieMiddleware is an HTTP middleware that creates or validates a user ID
// cookie. On each request it generates a new user ID (if the cookie is missing
// or invalid) or reuses the existing one. The resolved user ID is stored in the
// request context under UserIDContextKey.
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
	if _, err := rand.Read(b); err != nil {
		panic(err.Error())
	}
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
