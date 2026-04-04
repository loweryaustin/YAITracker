package handler

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (h *Handler) PostStartTimer(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	user := h.currentUser(r)
	issueID := r.FormValue("issue_id")
	if issueID == "" {
		http.Error(w, "issue_id required", http.StatusBadRequest)
		return
	}

	// Ensure a work session exists for the human
	sessionID := ""
	ws, _ := h.Store.GetActiveWorkSession(r.Context(), user.ID)
	if ws != nil {
		sessionID = ws.ID
	} else {
		ws, err := h.Store.CreateWorkSession(r.Context(), user.ID, "")
		if err != nil {
			http.Error(w, "could not create work session", http.StatusInternalServerError)
			return
		}
		sessionID = ws.ID
	}

	entry, err := h.Store.StartTimer(r.Context(), issueID, user.ID, "human", sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	issue, _ := h.Store.GetIssue(r.Context(), entry.IssueID)
	h.renderTimerWidget(w, r.Context(), issue, entry)
}

func (h *Handler) PostStopTimer(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	_, err := h.Store.StopTimer(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.renderTimerWidget(w, r.Context(), nil, nil)
}

func (h *Handler) PostManualTimeEntry(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	number := h.urlParamInt(r, "number")

	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	issue, err := h.Store.GetIssueByNumber(r.Context(), project.ID, number)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	user := h.currentUser(r)
	hours, _ := parseFloat(r.FormValue("hours"))
	description := r.FormValue("description")

	durationSecs := int64(hours * 3600)
	now := time.Now().UTC()
	startedAt := now.Add(-time.Duration(durationSecs) * time.Second)

	entry := &model.TimeEntry{
		IssueID:     issue.ID,
		UserID:      user.ID,
		Description: description,
		StartedAt:   startedAt,
		EndedAt:     &now,
		Duration:    &durationSecs,
	}

	h.Store.CreateManualTimeEntry(r.Context(), entry)

	entries, _ := h.Store.ListTimeEntries(r.Context(), issue.ID)
	h.renderTimeEntries(w, entries)
}

func (h *Handler) PatchTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := h.urlParam(r, "id")
	entry, err := h.Store.GetTimeEntry(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	if v := r.FormValue("description"); v != "" {
		entry.Description = v
	}
	if v := r.FormValue("hours"); v != "" {
		hours, _ := parseFloat(v)
		secs := int64(hours * 3600)
		entry.Duration = &secs
	}

	h.Store.UpdateTimeEntry(r.Context(), entry)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := h.urlParam(r, "id")
	h.Store.DeleteTimeEntry(r.Context(), id)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetTimesheet(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	weekStr := h.queryParam(r, "week")

	var weekStart time.Time
	if weekStr != "" {
		weekStart, _ = time.Parse("2006-01-02", weekStr)
	}
	if weekStart.IsZero() {
		now := time.Now()
		offset := int(now.Weekday()) - 1
		if offset < 0 {
			offset = 6
		}
		weekStart = now.AddDate(0, 0, -offset)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.AddDate(0, 0, 7)

	rows, err := h.Store.DB().QueryContext(r.Context(),
		`SELECT te.issue_id, te.started_at, te.duration,
		        i.number, i.title, i.project_id, p.key
		 FROM time_entries te
		 JOIN issues i ON te.issue_id = i.id
		 JOIN projects p ON i.project_id = p.id
		 WHERE te.user_id = ? AND te.started_at >= ? AND te.started_at < ? AND te.duration IS NOT NULL
		 ORDER BY i.project_id, i.number`,
		user.ID, weekStart, weekEnd)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type timesheetEntry struct {
		IssueID    string
		ProjectKey string
		Number     int
		Title      string
		Days       [7]int64
		Total      int64
	}

	entries := make(map[string]*timesheetEntry)
	var dailyTotal [7]int64
	var weekTotal int64

	for rows.Next() {
		var issueID, pKey string
		var startedAt time.Time
		var duration int64
		var num int
		var title, projectID string
		if err := rows.Scan(&issueID, &startedAt, &duration, &num, &title, &projectID, &pKey); err != nil {
			continue
		}
		dayIdx := int(startedAt.Weekday()) - 1
		if dayIdx < 0 {
			dayIdx = 6
		}

		e, ok := entries[issueID]
		if !ok {
			e = &timesheetEntry{IssueID: issueID, ProjectKey: pKey, Number: num, Title: title}
			entries[issueID] = e
		}
		e.Days[dayIdx] += duration
		e.Total += duration
		dailyTotal[dayIdx] += duration
		weekTotal += duration
	}

	type timesheetPageData struct {
		WeekStart  time.Time
		WeekEnd    time.Time
		PrevWeek   string
		NextWeek   string
		Entries    []*timesheetEntry
		DailyTotal [7]int64
		WeekTotal  int64
		DayNames   [7]string
	}

	entrySlice := make([]*timesheetEntry, 0, len(entries))
	for _, e := range entries {
		entrySlice = append(entrySlice, e)
	}

	pd := h.newPageData(r, "Timesheet", timesheetPageData{
		WeekStart:  weekStart,
		WeekEnd:    weekEnd,
		PrevWeek:   weekStart.AddDate(0, 0, -7).Format("2006-01-02"),
		NextWeek:   weekStart.AddDate(0, 0, 7).Format("2006-01-02"),
		Entries:    entrySlice,
		DailyTotal: dailyTotal,
		WeekTotal:  weekTotal,
		DayNames:   [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	})
	pd.ActiveNav = "timesheet"
	h.renderApp(w, r, "timesheet", timesheetTpl, pd)
}

func (h *Handler) renderTimerWidget(w http.ResponseWriter, ctx context.Context, issue *model.Issue, entry *model.TimeEntry) {
	timerTpl := template.Must(template.New("timer-widget").Parse(`
	{{if .Entry}}
	<div class="flex items-center gap-2" x-data="{ elapsed: {{.Elapsed}} }" x-init="setInterval(() => elapsed++, 1000)">
		<a href="/projects/{{.ProjectKey}}/issues/{{.Issue.Number}}" class="font-mono text-xs text-blue-600 hover:underline">
			{{.ProjectKey}}-{{.Issue.Number}}
		</a>
		<span class="text-sm font-mono" x-text="Math.floor(elapsed/3600) + 'h ' + Math.floor((elapsed%3600)/60) + 'm ' + (elapsed%60) + 's'"></span>
		<button hx-post="/time/stop" hx-target="#timer-widget" hx-swap="innerHTML"
				class="px-2 py-1 bg-red-50 text-red-600 rounded text-xs hover:bg-red-100">Stop</button>
	</div>
	{{else}}
	<span class="text-xs text-slate-400">No timer</span>
	{{end}}`))

	data := struct {
		Entry      *model.TimeEntry
		Issue      *model.Issue
		ProjectKey string
		Elapsed    int64
	}{Entry: entry, Issue: issue}

	if entry != nil && issue != nil {
		data.Elapsed = int64(time.Since(entry.StartedAt).Seconds())
		if p, err := h.Store.GetProjectByID(ctx, issue.ProjectID); err == nil {
			data.ProjectKey = p.Key
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	timerTpl.Execute(w, data)
}

func (h *Handler) renderTimeEntries(w http.ResponseWriter, entries []model.TimeEntry) {
	tpl := template.Must(template.New("time-entries").Funcs(template.FuncMap{
		"formatDuration": FormatDuration,
		"derefInt64":     func(i *int64) int64 { if i != nil { return *i }; return 0 },
	}).Parse(`
	{{range .}}
	<div class="flex items-center justify-between py-2 border-b border-slate-100 text-sm">
		<div class="flex items-center gap-3">
			<span>{{if .User}}{{.User.Name}}{{end}}</span>
			<span class="text-slate-400">{{.StartedAt.Format "Jan 2"}}</span>
		</div>
		<div class="flex items-center gap-3">
			<span class="font-mono">{{if .Duration}}{{formatDuration (derefInt64 .Duration)}}{{else}}running...{{end}}</span>
			<span class="text-slate-400 text-xs">{{.Description}}</span>
		</div>
	</div>
	{{end}}`))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tpl.Execute(w, entries)
}

var timesheetTpl = template.Must(template.New("timesheet").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$d := .Content}}
<h1 class="text-xl font-semibold text-slate-900 mb-6">Timesheet</h1>
<div class="flex items-center gap-4 mb-6">
    <a href="/time/sheet?week={{$d.PrevWeek}}" class="px-3 py-1 border border-slate-300 rounded hover:bg-slate-100 text-sm">&lt; Prev</a>
    <span class="font-medium">{{$d.WeekStart.Format "Jan 2"}} - {{$d.WeekEnd.Format "Jan 2, 2006"}}</span>
    <a href="/time/sheet?week={{$d.NextWeek}}" class="px-3 py-1 border border-slate-300 rounded hover:bg-slate-100 text-sm">Next &gt;</a>
</div>
<div class="bg-white rounded-lg border border-slate-200 overflow-hidden">
    <table class="w-full text-sm">
        <thead class="bg-slate-50 border-b border-slate-200">
            <tr>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Issue</th>
                {{range $d.DayNames}}
                <th class="text-center px-3 py-2 font-medium text-slate-500 w-16">{{.}}</th>
                {{end}}
                <th class="text-center px-3 py-2 font-medium text-slate-500 w-16">Total</th>
            </tr>
        </thead>
        <tbody>
            {{range $d.Entries}}
            <tr class="border-b border-slate-100">
                <td class="px-4 py-2">
                    <div class="font-mono text-xs text-slate-500">{{.ProjectKey}}-{{.Number}}</div>
                    <div class="text-sm truncate max-w-xs">{{.Title}}</div>
                </td>
                {{range .Days}}
                <td class="text-center px-3 py-2 text-xs">
                    {{if gt . 0}}{{formatDuration .}}{{else}}<span class="text-slate-300">-</span>{{end}}
                </td>
                {{end}}
                <td class="text-center px-3 py-2 text-xs font-medium">{{formatDuration .Total}}</td>
            </tr>
            {{end}}
            {{if not $d.Entries}}
            <tr><td colspan="9" class="px-4 py-8 text-center text-slate-400">No time tracked this week.</td></tr>
            {{end}}
        </tbody>
        <tfoot class="bg-slate-50 border-t border-slate-200">
            <tr>
                <td class="px-4 py-2 font-medium">Daily Total</td>
                {{range $d.DailyTotal}}
                <td class="text-center px-3 py-2 text-xs font-medium">
                    {{if gt . 0}}{{formatDuration .}}{{else}}-{{end}}
                </td>
                {{end}}
                <td class="text-center px-3 py-2 text-sm font-bold">{{formatDuration $d.WeekTotal}}</td>
            </tr>
        </tfoot>
    </table>
</div>
{{end}}
{{define "timer"}}
<span class="text-xs text-slate-400">No timer</span>
{{end}}`))
