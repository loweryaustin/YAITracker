package api

import (
	"net/http"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

func (a *API) ListProjects(w http.ResponseWriter, r *http.Request) {
	summaries, err := a.Store.ListProjectSummaries(r.Context())
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not list projects")
		return
	}
	if summaries == nil {
		summaries = []model.ProjectSummary{}
	}
	a.jsonOK(w, summaries)
}

func (a *API) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key         string   `json:"key"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		TargetDate  string   `json:"target_date"`
		BudgetHours *float64 `json:"budget_hours"`
		Tags        []string `json:"tags"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user := a.currentUser(r)
	p := &model.Project{
		Key:         strings.ToUpper(req.Key),
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		BudgetHours: req.BudgetHours,
		CreatedBy:   user.ID,
	}

	if req.TargetDate != "" {
		t, err := parseDate(req.TargetDate)
		if err == nil {
			p.TargetDate = &t
		}
	}

	if p.Key == "" || p.Name == "" {
		a.jsonError(w, http.StatusBadRequest, "validation_error", "Key and name are required")
		return
	}

	if err := a.Store.CreateProject(r.Context(), p); err != nil {
		a.jsonError(w, http.StatusConflict, "conflict", err.Error())
		return
	}

	for _, tag := range req.Tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag != "" {
			a.Store.AddProjectTag(r.Context(), p.ID, tag, "")
		}
	}

	p.Tags, _ = a.Store.GetProjectTags(r.Context(), p.ID)
	a.jsonCreated(w, p)
}

func (a *API) GetProject(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	a.jsonOK(w, p)
}

func (a *API) UpdateProject(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		Status      *string  `json:"status"`
		TargetDate  *string  `json:"target_date"`
		BudgetHours *float64 `json:"budget_hours"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.TargetDate != nil {
		t, err := parseDate(*req.TargetDate)
		if err == nil {
			p.TargetDate = &t
		}
	}
	if req.BudgetHours != nil {
		p.BudgetHours = req.BudgetHours
	}

	a.Store.UpdateProject(r.Context(), p)
	a.jsonOK(w, p)
}

func (a *API) DeleteProject(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	if err := a.Store.DeleteProject(r.Context(), p.ID); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) GetMembers(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	members, _ := a.Store.GetProjectMembers(r.Context(), p.ID)
	if members == nil {
		members = []model.ProjectMember{}
	}
	a.jsonOK(w, members)
}

func (a *API) AddMember(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}

	a.Store.AddProjectMember(r.Context(), p.ID, req.UserID, req.Role)
	w.WriteHeader(http.StatusCreated)
}
