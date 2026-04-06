package api

import (
	"net/http"

	"yaitracker.com/loweryaustin/internal/model"
)

func (a *API) ListComments(w http.ResponseWriter, r *http.Request) {
	issueID := a.urlParam(r, "id")
	comments, err := a.Store.ListComments(r.Context(), issueID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not list comments")
		return
	}
	if comments == nil {
		comments = []model.Comment{}
	}
	a.jsonOK(w, comments)
}

func (a *API) CreateComment(w http.ResponseWriter, r *http.Request) {
	issueID := a.urlParam(r, "id")
	var req struct {
		Body string `json:"body"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Body == "" {
		a.jsonError(w, http.StatusBadRequest, "validation_error", "Body is required")
		return
	}

	user := a.currentUser(r)
	c := &model.Comment{
		IssueID:  issueID,
		AuthorID: user.ID,
		Body:     req.Body,
	}
	if err := a.Store.CreateComment(r.Context(), c); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonCreated(w, c)
}

func (a *API) UpdateComment(w http.ResponseWriter, r *http.Request) {
	id := a.urlParam(r, "id")
	c, err := a.Store.GetComment(r.Context(), id)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Comment not found")
		return
	}

	user := a.currentUser(r)
	if c.AuthorID != user.ID && !user.IsAdmin() {
		a.jsonError(w, http.StatusForbidden, "forbidden", "Not authorized")
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	c.Body = req.Body
	if err := a.Store.UpdateComment(r.Context(), c); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, c)
}

func (a *API) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id := a.urlParam(r, "id")
	c, err := a.Store.GetComment(r.Context(), id)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Comment not found")
		return
	}

	user := a.currentUser(r)
	if c.AuthorID != user.ID && !user.IsAdmin() {
		a.jsonError(w, http.StatusForbidden, "forbidden", "Not authorized")
		return
	}

	if err := a.Store.DeleteComment(r.Context(), id); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
