package auth

import (
	"net/http"
	"strings"

	"yaitracker.com/loweryaustin/internal/store"
)

// BearerTokenMiddleware authenticates API requests via OAuth2 bearer tokens.
func BearerTokenMiddleware(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing_token","message":"Authorization header required"}`, http.StatusUnauthorized)
				return
			}

			accessToken := strings.TrimPrefix(authHeader, "Bearer ")
			tok, err := st.GetOAuthTokenByAccess(r.Context(), accessToken)
			if err != nil {
				http.Error(w, `{"error":"invalid_token","message":"Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			user, err := st.GetUserByID(r.Context(), tok.UserID)
			if err != nil {
				http.Error(w, `{"error":"user_not_found","message":"Token user not found"}`, http.StatusUnauthorized)
				return
			}

			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
