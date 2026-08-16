package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSecret = "test-secret-key-for-signing-cookies-only"

func TestInit(t *testing.T) {
	Init(testSecret)

	if string(key) != testSecret {
		t.Errorf("key = %q, want %q", string(key), testSecret)
	}
}

func TestInit_EmptyKey(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Init() panicked with empty key: %v", r)
		}
	}()

	Init("")
	// Init should not panic with empty key
}

func TestNewID(t *testing.T) {
	Init(testSecret)

	id1 := newID()
	id2 := newID()

	if len(id1) == 0 {
		t.Error("newID() returned empty string")
	}

	// IDs should be different (16 bytes = 32 hex chars)
	if len(id1) != 32 {
		t.Errorf("newID() returned ID of length %d, want 32", len(id1))
	}

	if id1 == id2 {
		t.Error("newID() returned the same value twice")
	}
}

func TestMac(t *testing.T) {
	Init(testSecret)

	mac1 := mac("user123")
	mac2 := mac("user123")
	mac3 := mac("user456")

	if len(mac1) == 0 {
		t.Error("mac() returned empty string")
	}

	// Same input should produce same MAC
	if mac1 != mac2 {
		t.Error("mac() returned different values for the same input")
	}

	// Different input should produce different MAC
	if mac1 == mac3 {
		t.Error("mac() returned same value for different inputs")
	}
}

func TestSignAndVerify(t *testing.T) {
	Init(testSecret)

	userID := "abc123"
	signed := sign(userID)

	// Verify returns true for valid signature
	if !verify(signed) {
		t.Error("verify() returned false for valid signature")
	}

	// Verify returns false for tampered signature
	tampered := "abc123|tampered"
	if verify(tampered) {
		t.Error("verify() returned true for tampered signature")
	}

	// Verify returns false for empty string
	if verify("") {
		t.Error("verify() returned true for empty string")
	}

	// Verify returns false for string without separator
	if verify("noSignature") {
		t.Error("verify() returned true for string without separator")
	}
}

func TestGetOrCreateUser_NewRequest(t *testing.T) {
	Init(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	userID := getOrCreateUser(req, rec)

	if userID == "" {
		t.Error("getOrCreateUser() returned empty userID for new request")
	}
}

func TestGetOrCreateUser_InvalidCookie(t *testing.T) {
	Init(testSecret)

	// Create request with invalid cookie (wrong signature)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "invalid|invalid"})

	rec := httptest.NewRecorder()

	userID := getOrCreateUser(req, rec)

	if userID == "" {
		t.Error("getOrCreateUser() returned empty userID for invalid cookie")
	}
}

func TestGetOrCreateUser_ValidCookie(t *testing.T) {
	Init(testSecret)

	// Create a valid cookie
	userID := newID()
	signed := sign(userID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: signed})

	rec := httptest.NewRecorder()

	result := getOrCreateUser(req, rec)

	if result != userID {
		t.Errorf("getOrCreateUser() returned %q, want %q", result, userID)
	}
}

func TestGetOrCreateUser_MalformedCookie(t *testing.T) {
	Init(testSecret)

	// Cookie without signature separator
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "user-only-no-pipe"})

	rec := httptest.NewRecorder()

	userID := getOrCreateUser(req, rec)

	if userID == "" {
		t.Error("getOrCreateUser() returned empty userID for malformed cookie")
	}

	// Should generate a new ID since verification fails
	if userID == "user-only-no-pipe" {
		t.Error("getOrCreateUser() should generate new ID for malformed cookie, not reuse it")
	}
}

func TestCookieMiddleware_NewRequest(t *testing.T) {
	Init(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user ID is in context
		ctxUserID := r.Context().Value(UserIDContextKey)
		if ctxUserID == nil {
			t.Error("UserIDContextKey not found in context")
		}
	})

	middleware := CookieMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	// Check that cookie was set
	res := rec.Result()
	defer res.Body.Close()
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Error("CookieMiddleware did not set any cookies")
	}

	found := false
	for _, cookie := range cookies {
		if cookie.Name == CookieName {
			found = true
			if cookie.Path != "/" {
				t.Errorf("cookie path = %q, want %q", cookie.Path, "/")
			}
			if !cookie.HttpOnly {
				t.Error("cookie HttpOnly is not set")
			}
		}
	}

	if !found {
		t.Error("CookieMiddleware did not set the userID cookie")
	}
}

func TestCookieMiddleware_ValidCookie(t *testing.T) {
	Init(testSecret)

	// Create a valid signed cookie
	userID := newID()
	signed := sign(userID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: signed})
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID := r.Context().Value(UserIDContextKey).(string)
		if ctxUserID != userID {
			t.Errorf("context userID = %q, want %q", ctxUserID, userID)
		}
	})

	middleware := CookieMiddleware(handler)
	middleware.ServeHTTP(rec, req)
}

func TestCookieMiddleware_InvalidCookie(t *testing.T) {
	Init(testSecret)

	// Create request with invalid cookie
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "invalid|signature"})
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID := r.Context().Value(UserIDContextKey).(string)
		// Should generate new ID, not use the invalid one
		if ctxUserID == "invalid" {
			t.Error("context userID should not be the invalid cookie value")
		}
	})

	middleware := CookieMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	// Should have set a new valid cookie
	res := rec.Result()
	defer res.Body.Close()
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Error("CookieMiddleware did not set any cookies for invalid request")
	}
}

func TestCookieMiddleware_ContextPropagation(t *testing.T) {
	Init(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	var capturedUserID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Context().Value(UserIDContextKey).(string)
	})

	middleware := CookieMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	if capturedUserID == "" {
		t.Error("userID was not propagated to context")
	}
}

func TestCookieMiddleware_ContextKeyExists(t *testing.T) {
	Init(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	keyFound := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, keyFound = r.Context().Value(UserIDContextKey).(string)
	})

	middleware := CookieMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	if !keyFound {
		t.Error("UserIDContextKey was not set in context")
	}
}
