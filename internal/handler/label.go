package handler

import (
	"net/http"

	"yaitracker.com/loweryaustin/internal/model"
)

func (h *Handler) PostLabel(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	label := &model.Label{
		ProjectID: project.ID,
		Name:      r.FormValue("name"),
		Color:     r.FormValue("color"),
	}

	if label.Name == "" || label.Color == "" {
		http.Error(w, "Name and color required", http.StatusBadRequest)
		return
	}

	if err := h.Store.CreateLabel(r.Context(), label); err != nil {
		http.Error(w, "Could not create label", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+key+"/settings", http.StatusSeeOther)
}

func (h *Handler) PatchLabel(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	id := h.urlParam(r, "id")
	label, err := h.Store.GetLabel(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	if v := r.FormValue("name"); v != "" {
		label.Name = v
	}
	if v := r.FormValue("color"); v != "" {
		label.Color = v
	}

	if err := h.Store.UpdateLabel(r.Context(), label); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	id := h.urlParam(r, "id")
	if err := h.Store.DeleteLabel(r.Context(), id); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
