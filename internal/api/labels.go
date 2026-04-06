package api

import (
	"net/http"

	"yaitracker.com/loweryaustin/internal/model"
)

func (a *API) ListLabels(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	labels, err := a.Store.ListLabels(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if labels == nil {
		labels = []model.Label{}
	}
	a.jsonOK(w, labels)
}

func (a *API) CreateLabel(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	label := &model.Label{ProjectID: p.ID, Name: req.Name, Color: req.Color}
	if err := a.Store.CreateLabel(r.Context(), label); err != nil {
		a.jsonError(w, http.StatusConflict, "conflict", err.Error())
		return
	}
	a.jsonCreated(w, label)
}

func (a *API) UpdateLabel(w http.ResponseWriter, r *http.Request) {
	id := a.urlParam(r, "id")
	label, err := a.Store.GetLabel(r.Context(), id)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Label not found")
		return
	}

	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name != nil {
		label.Name = *req.Name
	}
	if req.Color != nil {
		label.Color = *req.Color
	}
	if err := a.Store.UpdateLabel(r.Context(), label); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, label)
}

func (a *API) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	id := a.urlParam(r, "id")
	if err := a.Store.DeleteLabel(r.Context(), id); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
