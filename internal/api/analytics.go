package api

import (
	"net/http"
	"strconv"
	"strings"
)

func (a *API) GetVelocity(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	weeks := a.queryParamInt(r, "weeks", 8)
	velocity, err := a.Store.GetVelocity(r.Context(), p.ID, weeks)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, velocity)
}

func (a *API) GetCycleTime(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	stats, err := a.Store.GetCycleTimeStats(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, stats)
}

func (a *API) GetEstimates(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	report, err := a.Store.GetEstimationReport(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, report)
}

func (a *API) GetTimeSpent(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	data, err := a.Store.GetTimeByType(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, data)
}

func (a *API) GetHealth(w http.ResponseWriter, r *http.Request) {
	key := a.urlParam(r, "key")
	p, err := a.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Project not found")
		return
	}
	health, err := a.Store.GetProjectHealth(r.Context(), p.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, health)
}

func (a *API) Compare(w http.ResponseWriter, r *http.Request) {
	groupBy := a.queryParam(r, "group_by")
	if groupBy == "" {
		a.jsonError(w, http.StatusBadRequest, "validation_error", "group_by parameter required")
		return
	}
	comparisons, err := a.Store.CompareByTag(r.Context(), groupBy)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, comparisons)
}

func (a *API) Predict(w http.ResponseWriter, r *http.Request) {
	tagsStr := a.queryParam(r, "tags")
	pointsStr := a.queryParam(r, "points")
	points, _ := strconv.Atoi(pointsStr) //nolint:errcheck // returns 0 on invalid input, checked below

	if tagsStr == "" || points <= 0 {
		a.jsonError(w, http.StatusBadRequest, "validation_error", "tags and points parameters required")
		return
	}

	var tags []string
	for _, t := range strings.Split(tagsStr, ",") {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			tags = append(tags, t)
		}
	}

	pred, err := a.Store.PredictNewProject(r.Context(), tags, points)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonOK(w, pred)
}
