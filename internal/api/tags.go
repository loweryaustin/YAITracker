package api

import (
	"net/http"
	"strings"
)

func (a *API) ListProjectTags(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	tags, err := a.Store.GetProjectTags(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, tags)
}

func (a *API) AddProjectTag(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	var req struct {
		Tag       string `json:"tag"`
		GroupName string `json:"group_name"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	tag := strings.TrimSpace(strings.ToLower(req.Tag))
	if tag == "" {
		a.jsonError(w, http.StatusBadRequest, "validation_error", "Tag is required")
		return
	}

	if err := a.Store.AddProjectTag(r.Context(), p.ID, tag, req.GroupName); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *API) RemoveProjectTag(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	tag := a.urlParam(r, "tag")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	if err := a.Store.RemoveProjectTag(r.Context(), p.ID, tag); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListAllTags(w http.ResponseWriter, r *http.Request) {
	tags, err := a.Store.ListAllTags(r.Context())
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, tags)
}

func (a *API) ListTagGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := a.Store.ListTagGroups(r.Context())
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, groups)
}
