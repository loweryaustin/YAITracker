package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
)

type Handler struct {
	Store  *store.Store
	Secret string
	Secure bool // true if behind HTTPS
}

func New(st *store.Store, secret string) *Handler {
	return &Handler{Store: st, Secret: secret}
}

func (h *Handler) currentUser(r *http.Request) *model.User {
	return auth.UserFromContext(r.Context())
}

func (h *Handler) urlParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

func (h *Handler) urlParamInt(r *http.Request, name string) int { //nolint:unparam // generic helper, only "number" currently
	v, _ := strconv.Atoi(chi.URLParam(r, name)) //nolint:errcheck // returns 0 on invalid input
	return v
}

func (h *Handler) queryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

func (h *Handler) queryParamInt(r *http.Request, name, defaultVal string) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		v = defaultVal
	}
	n, _ := strconv.Atoi(v) //nolint:errcheck // returns 0 on invalid input
	return n
}

func (h *Handler) clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
