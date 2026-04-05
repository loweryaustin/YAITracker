package api

import (
	"database/sql"
	"net/http"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (a *API) StartTimer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID string `json:"issue_id"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user := a.currentUser(r)

	// Ensure a work session exists for the human
	sessionID := ""
	ws, _ := a.Store.GetActiveWorkSession(r.Context(), user.ID)
	if ws != nil {
		sessionID = ws.ID
	} else {
		ws, err := a.Store.CreateWorkSession(r.Context(), user.ID, "")
		if err != nil {
			a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not create work session")
			return
		}
		sessionID = ws.ID
	}

	entry, err := a.Store.StartTimer(r.Context(), req.IssueID, user.ID, "human", sessionID, "", "")
	if err != nil {
		a.jsonError(w, http.StatusConflict, "conflict", err.Error())
		return
	}
	a.jsonCreated(w, entry)
}

func (a *API) StopTimer(w http.ResponseWriter, r *http.Request) {
	user := a.currentUser(r)
	entry, err := a.Store.StopTimer(r.Context(), user.ID)
	if err != nil {
		a.jsonError(w, http.StatusBadRequest, "no_timer", err.Error())
		return
	}
	a.jsonOK(w, entry)
}

func (a *API) GetActiveTimer(w http.ResponseWriter, r *http.Request) {
	user := a.currentUser(r)
	entry, err := a.Store.GetActiveTimer(r.Context(), user.ID)
	if err != nil || entry == nil {
		a.jsonOK(w, nil)
		return
	}
	a.jsonOK(w, entry)
}

func (a *API) ListTimeEntries(w http.ResponseWriter, r *http.Request) {
	issueID := a.urlParam(r, "id")
	entries, err := a.Store.ListTimeEntries(r.Context(), issueID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not list time entries")
		return
	}
	if entries == nil {
		entries = []model.TimeEntry{}
	}
	a.jsonOK(w, entries)
}

func (a *API) CreateTimeEntry(w http.ResponseWriter, r *http.Request) {
	issueID := a.urlParam(r, "id")

	var req struct {
		Hours       float64 `json:"hours"`
		Description string  `json:"description"`
		Date        string  `json:"date"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user := a.currentUser(r)
	durationSecs := int64(req.Hours * 3600)
	now := time.Now().UTC()

	var startedAt time.Time
	if req.Date != "" {
		startedAt, _ = time.Parse("2006-01-02", req.Date)
	}
	if startedAt.IsZero() {
		startedAt = now.Add(-time.Duration(durationSecs) * time.Second)
	}

	entry := &model.TimeEntry{
		IssueID:     issueID,
		UserID:      user.ID,
		Description: req.Description,
		StartedAt:   startedAt,
		EndedAt:     &now,
		Duration:    &durationSecs,
	}

	if err := a.Store.CreateManualTimeEntry(r.Context(), entry); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.jsonCreated(w, entry)
}

func (a *API) UpdateTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := a.urlParam(r, "id")
	entry, err := a.Store.GetTimeEntry(r.Context(), id)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", "Time entry not found")
		return
	}

	var req struct {
		Hours       *float64 `json:"hours"`
		Description *string  `json:"description"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Hours != nil {
		secs := int64(*req.Hours * 3600)
		entry.Duration = &secs
	}
	if req.Description != nil {
		entry.Description = *req.Description
	}

	a.Store.UpdateTimeEntry(r.Context(), entry)
	a.jsonOK(w, entry)
}

func (a *API) DeleteTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := a.urlParam(r, "id")
	a.Store.DeleteTimeEntry(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) GetTimesheet(w http.ResponseWriter, r *http.Request) {
	// Simplified timesheet - return entries for date range
	user := a.currentUser(r)
	from := a.queryParam(r, "from")
	to := a.queryParam(r, "to")

	var startDate, endDate time.Time
	if from != "" {
		startDate, _ = time.Parse("2006-01-02", from)
	}
	if to != "" {
		endDate, _ = time.Parse("2006-01-02", to)
	}
	if startDate.IsZero() {
		now := time.Now()
		offset := int(now.Weekday()) - 1
		if offset < 0 {
			offset = 6
		}
		startDate = now.AddDate(0, 0, -offset)
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	}
	if endDate.IsZero() {
		endDate = startDate.AddDate(0, 0, 7)
	}

	rows, err := a.Store.DB().QueryContext(r.Context(),
		`SELECT te.id, te.issue_id, te.user_id, te.description, te.actor_type, te.mcp_actor_id,
		        te.started_at, te.ended_at, te.duration, te.source, te.created_at, te.updated_at
		 FROM time_entries te
		 WHERE te.user_id = ? AND te.started_at >= ? AND te.started_at < ? AND te.duration IS NOT NULL
		 ORDER BY te.started_at`,
		user.ID, startDate, endDate)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not load timesheet")
		return
	}
	defer rows.Close()

	var entries []model.TimeEntry
	for rows.Next() {
		var e model.TimeEntry
		var desc, mcp sql.NullString
		var endedAt sql.NullTime
		var duration int64
		if err := rows.Scan(&e.ID, &e.IssueID, &e.UserID, &desc, &e.ActorType, &mcp,
			&e.StartedAt, &endedAt, &duration, &e.Source, &e.CreatedAt, &e.UpdatedAt); err != nil {
			a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not load timesheet")
			return
		}
		if desc.Valid {
			e.Description = desc.String
		}
		if mcp.Valid {
			e.McpActorID = mcp.String
		}
		e.EndedAt = nil
		if endedAt.Valid {
			t := endedAt.Time
			e.EndedAt = &t
		}
		d := duration
		e.Duration = &d
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []model.TimeEntry{}
	}

	a.jsonOK(w, map[string]interface{}{
		"from":    startDate.Format("2006-01-02"),
		"to":      endDate.Format("2006-01-02"),
		"entries": entries,
	})
}
