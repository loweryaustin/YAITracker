package handler

import (
	"encoding/json"
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

func (h *Handler) urlParamInt(r *http.Request, name string) int {
	v, _ := strconv.Atoi(chi.URLParam(r, name))
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
	n, _ := strconv.Atoi(v)
	return n
}

func (h *Handler) clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) htmlError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}
