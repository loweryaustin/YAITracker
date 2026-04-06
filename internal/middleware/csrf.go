package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	CSRFCookieName = "yaitracker_csrf"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFFormField  = "_csrf"
)

type csrfCtxKey struct{}

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			r = ensureCSRFCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		token := r.Header.Get(CSRFHeaderName)
		if token == "" {
			token = r.FormValue(CSRFFormField) //nolint:gosec // G120: body size limited by BodyLimit middleware in routes //nolint:gosec // body-size limits belong to outer handler; CSRF reads one field
		}

		if token == "" || token != cookie.Value {
			http.Error(w, "CSRF token invalid", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ensureCSRFCookie sets the CSRF cookie if absent and stashes the token in
// the request context so GetCSRFToken works even on the first page load
// (before the browser has stored the Set-Cookie response).
func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) *http.Request {
	if c, err := r.Cookie(CSRFCookieName); err == nil && c.Value != "" {
		return r.WithContext(context.WithValue(r.Context(), csrfCtxKey{}, c.Value))
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JS needs to read this for htmx hx-headers
		SameSite: http.SameSiteLaxMode,
	})
	return r.WithContext(context.WithValue(r.Context(), csrfCtxKey{}, token))
}

// GetCSRFToken returns the CSRF token for the current request, preferring
// the context value (set by the middleware) over the raw cookie.
func GetCSRFToken(r *http.Request) string {
	if tok, ok := r.Context().Value(csrfCtxKey{}).(string); ok && tok != "" {
		return tok
	}
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
