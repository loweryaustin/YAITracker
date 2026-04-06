package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRF_FirstLoadTokenInForm(t *testing.T) {
	t.Parallel()

	handler := CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := GetCSRFToken(r)
		if tok == "" {
			t.Fatal("expected non-empty CSRF token on first GET")
		}
		w.Write([]byte(tok)) //nolint:errcheck
	}))

	req := httptest.NewRequest("GET", "/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rec.Code)
	}

	// The token rendered in the body must match the Set-Cookie value.
	bodyToken := rec.Body.String()
	cookies := rec.Result().Cookies()
	var cookieToken string
	for _, c := range cookies {
		if c.Name == CSRFCookieName {
			cookieToken = c.Value
		}
	}
	if cookieToken == "" {
		t.Fatal("CSRF cookie not set on first GET")
	}
	if bodyToken != cookieToken {
		t.Fatalf("body token %q != cookie token %q", bodyToken, cookieToken)
	}
}

func TestCSRF_PostWithValidToken(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRF(inner)

	// First GET to obtain a token.
	getReq := httptest.NewRequest("GET", "/login", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFCookieName {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("no CSRF cookie from GET")
	}

	// POST with cookie + form field.
	body := strings.NewReader(CSRFFormField + "=" + token)
	postReq := httptest.NewRequest("POST", "/login", body)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})

	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want 200; body: %s", postRec.Code, postRec.Body.String())
	}
}

func TestCSRF_PostWithoutCookie(t *testing.T) {
	t.Parallel()

	handler := CSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.NewReader(CSRFFormField + "=sometoken")
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST without cookie: status = %d, want 403", rec.Code)
	}
}

func TestCSRF_PostWithMismatchedToken(t *testing.T) {
	t.Parallel()

	handler := CSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.NewReader(CSRFFormField + "=wrong")
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "correct"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST with mismatched token: status = %d, want 403", rec.Code)
	}
}

func TestCSRF_PostWithHeaderToken(t *testing.T) {
	t.Parallel()

	handler := CSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest("GET", "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFCookieName {
			token = c.Value
		}
	}

	req := httptest.NewRequest("POST", "/action", nil)
	req.Header.Set(CSRFHeaderName, token)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST with header token: status = %d, want 200", rec.Code)
	}
}

func TestCSRF_ExistingCookieNotOverwritten(t *testing.T) {
	t.Parallel()

	handler := CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := GetCSRFToken(r)
		w.Write([]byte(tok)) //nolint:errcheck
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "existing-token"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "existing-token" {
		t.Fatalf("expected existing token, got %q", rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == CSRFCookieName {
			t.Fatal("should not re-set cookie when one already exists")
		}
	}
}
