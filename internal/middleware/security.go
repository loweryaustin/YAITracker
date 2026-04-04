package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

type ctxKeyNonce struct{}

func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// GetCSPNonce retrieves the per-request CSP nonce from the context.
func GetCSPNonce(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyNonce{}).(string); ok {
		return v
	}
	return ""
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := generateNonce()
		ctx := context.WithValue(r.Context(), ctxKeyNonce{}, nonce)
		r = r.WithContext(ctx)

		w.Header().Set("Content-Security-Policy", fmt.Sprintf(
			"default-src 'self'; script-src 'self' 'nonce-%s' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'",
			nonce))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
