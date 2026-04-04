package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	CSRFCookieName = "yaitracker_csrf"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFFormField  = "_csrf"
)

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read-only methods don't need CSRF
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			ensureCSRFCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		// Check header first, then form field
		token := r.Header.Get(CSRFHeaderName)
		if token == "" {
			token = r.FormValue(CSRFFormField)
		}

		if token == "" || token != cookie.Value {
			http.Error(w, "CSRF token invalid", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(CSRFCookieName); err == nil {
		return
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
}

func GetCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
