package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
)

type API struct {
	Store *store.Store
}

func New(st *store.Store) *API {
	return &API{Store: st}
}

func (a *API) currentUser(r *http.Request) *model.User {
	return auth.UserFromContext(r.Context())
}

func (a *API) urlParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

func (a *API) urlParamInt(r *http.Request, name string) int {
	v, _ := strconv.Atoi(chi.URLParam(r, name))
	return v
}

func (a *API) queryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

func (a *API) queryParamInt(r *http.Request, name string, def int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (a *API) jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (a *API) jsonCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func (a *API) jsonError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiError{Error: code, Message: message})
}

func (a *API) decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func (a *API) clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
