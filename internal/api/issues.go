package api

import (
	"fmt"
	"net/http"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

func (a *API) ListIssues(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	filter := model.IssueFilter{
		ProjectID: p.ID,
		Query:     a.queryParam(r, "q"),
		SortBy:    a.queryParam(r, "sort"),
		SortDir:   a.queryParam(r, "dir"),
		Limit:     min(a.queryParamInt(r, "limit", 25), 100),
		Offset:    a.queryParamInt(r, "offset", 0),
	}

	if s := a.queryParam(r, "status"); s != "" {
		filter.Status = strings.Split(s, ",")
	}
	if t := a.queryParam(r, "type"); t != "" {
		filter.Type = strings.Split(t, ",")
	}
	if p := a.queryParam(r, "priority"); p != "" {
		filter.Priority = strings.Split(p, ",")
	}
	if aid := a.queryParam(r, "assignee_id"); aid != "" {
		filter.AssigneeID = aid
	}

	issues, total, err := a.Store.ListIssues(r.Context(), filter)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not list issues")
		return
	}

	if issues == nil {
		issues = []model.Issue{}
	}
	a.jsonOK(w, map[string]interface{}{
		"issues": issues,
		"total":  total,
	})
}

func (a *API) CreateIssue(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	var req struct {
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		Type           string   `json:"type"`
		Status         string   `json:"status"`
		Priority       string   `json:"priority"`
		AssigneeID     *string  `json:"assignee_id"`
		ParentID       *string  `json:"parent_id"`
		StoryPoints    *int     `json:"story_points"`
		EstimatedHours *float64 `json:"estimated_hours"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Title == "" {
		a.jsonError(w, http.StatusBadRequest, "validation_error", "Title is required")
		return
	}

	user := a.currentUser(r)
	issue := &model.Issue{
		ProjectID:      p.ID,
		Title:          req.Title,
		Description:    req.Description,
		Type:           req.Type,
		Status:         req.Status,
		Priority:       req.Priority,
		AssigneeID:     req.AssigneeID,
		ParentID:       req.ParentID,
		StoryPoints:    req.StoryPoints,
		EstimatedHours: req.EstimatedHours,
		ReporterID:     user.ID,
	}

	if issue.Type == "" {
		issue.Type = "task"
	}
	if issue.Status == "" {
		issue.Status = "backlog"
	}
	if issue.Priority == "" {
		issue.Priority = "none"
	}

	if err := a.Store.CreateIssue(r.Context(), issue); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	a.jsonCreated(w, issue)
}

func (a *API) GetIssue(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	number := a.urlParamInt(r, "number")

	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	issue, err := a.Store.GetIssueByNumber(r.Context(), p.ID, number)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Issue not found")
		return
	}

	a.jsonOK(w, issue)
}

func (a *API) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	number := a.urlParamInt(r, "number")

	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	issue, err := a.Store.GetIssueByNumber(r.Context(), p.ID, number)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Issue not found")
		return
	}

	var req struct {
		Title          *string  `json:"title"`
		Description    *string  `json:"description"`
		Type           *string  `json:"type"`
		Status         *string  `json:"status"`
		Priority       *string  `json:"priority"`
		AssigneeID     *string  `json:"assignee_id"`
		ParentID       *string  `json:"parent_id"`
		StoryPoints    *int     `json:"story_points"`
		EstimatedHours *float64 `json:"estimated_hours"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Description != nil {
		issue.Description = *req.Description
	}
	if req.Type != nil {
		issue.Type = *req.Type
	}
	if req.Priority != nil {
		issue.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		issue.AssigneeID = req.AssigneeID
	}
	if req.ParentID != nil {
		issue.ParentID = req.ParentID
	}
	if req.StoryPoints != nil {
		issue.StoryPoints = req.StoryPoints
	}
	if req.EstimatedHours != nil {
		issue.EstimatedHours = req.EstimatedHours
	}

	if req.Status != nil && *req.Status != issue.Status {
		if err := a.Store.UpdateIssueStatus(r.Context(), issue.ID, *req.Status); err != nil {
			a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		issue.Status = *req.Status
	}

	if err := a.Store.UpdateIssue(r.Context(), issue); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, issue)
}

func (a *API) DeleteIssue(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	number := a.urlParamInt(r, "number")

	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	issue, err := a.Store.GetIssueByNumber(r.Context(), p.ID, number)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Issue not found")
		return
	}

	if err := a.Store.DeleteIssue(r.Context(), issue.ID); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) GetBoard(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	columns, err := a.Store.ListIssuesByStatus(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not load board")
		return
	}
	a.jsonOK(w, columns)
}

func (a *API) MoveIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID   string  `json:"issue_id"`
		Status    string  `json:"status"`
		SortOrder float64 `json:"sort_order"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := a.Store.MoveIssue(r.Context(), req.IssueID, req.Status, req.SortOrder); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"ok":true}`) //nolint:errcheck // response write error is not recoverable
}
