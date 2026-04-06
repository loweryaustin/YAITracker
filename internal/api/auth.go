package api

import (
	"net/http"
	"time"

	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/model"
)

func (a *API) PostToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		ClientName string `json:"client_name"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user, err := a.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		a.jsonError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	if user.IsLocked() {
		a.jsonError(w, http.StatusTooManyRequests, "account_locked", "Account is temporarily locked")
		return
	}

	if !auth.CheckPassword(user.Password, req.Password) {
		count, err := a.Store.IncrementFailedAttempts(r.Context(), user.ID)
		if err != nil {
			a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not update login attempts")
			return
		}
		if count >= 10 {
			if err := a.Store.LockUser(r.Context(), user.ID, time.Now().UTC().Add(15*time.Minute)); err != nil {
				a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not lock account")
				return
			}
		}
		a.jsonError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	if err := a.Store.ResetFailedAttempts(r.Context(), user.ID); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not reset login attempts")
		return
	}

	token, err := a.Store.CreateOAuthToken(r.Context(), user.ID, req.ClientName)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not create token")
		return
	}

	a.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit logging
		EntityType: "auth", UserID: user.ID, Action: "token_issued",
		NewValue: req.ClientName, IPAddress: a.clientIP(r),
	})

	a.jsonOK(w, map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    "bearer",
		"expires_in":    3600,
	})
}

func (a *API) PostRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	token, err := a.Store.RefreshOAuthToken(r.Context(), req.RefreshToken)
	if err != nil {
		a.jsonError(w, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}

	a.jsonOK(w, map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    "bearer",
		"expires_in":    3600,
	})
}

func (a *API) GetMe(w http.ResponseWriter, r *http.Request) {
	user := a.currentUser(r)
	a.jsonOK(w, user)
}

func (a *API) DeleteToken(w http.ResponseWriter, r *http.Request) {
	// Revoke the current token
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 {
		accessToken := authHeader[7:]
		tok, err := a.Store.GetOAuthTokenByAccess(r.Context(), accessToken)
		if err == nil {
			a.Store.DeleteOAuthToken(r.Context(), tok.ID) //nolint:errcheck // best-effort revocation
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
