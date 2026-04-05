package handler

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
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

	if _, err := h.Store.StartTimer(r.Context(), issueID, user.ID, "human", sessionID, "", ""); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	h.renderSessionBanner(w, r)
}

func (h *Handler) PostStopTimer(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if _, err := h.Store.StopTimer(r.Context(), user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.renderSessionBanner(w, r)
}

func (h *Handler) PostSessionStart(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if _, err := h.Store.CreateWorkSession(r.Context(), user.ID, ""); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	h.renderSessionBanner(w, r)
}

func (h *Handler) PostSessionEnd(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	if _, err := h.Store.EndWorkSession(r.Context(), user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.renderSessionBanner(w, r)
}

func (h *Handler) GetSessionBanner(w http.ResponseWriter, r *http.Request) {
	h.renderSessionBanner(w, r)
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

type timeHubData struct {
	ActiveTimers []model.TimeEntry
	DailySummary *store.DailySummary
	Session      *model.WorkSession
	SessionSecs  int64
	Sessions     []model.WorkSession
	WeekStart    time.Time
	WeekEnd      time.Time
	PrevWeek     string
	NextWeek     string
	TSEntries    []*timesheetEntry
	DailyTotal   [7]int64
	WeekTotal    int64
	DayNames     [7]string
}

type timesheetEntry struct {
	IssueID    string
	ProjectKey string
	Number     int
	Title      string
	ActorType  string
	McpActorID string
	Days       [7]int64
	Total      int64
}

func (h *Handler) GetTimeHub(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	ctx := r.Context()

	activeTimers, _ := h.Store.GetActiveTimersWithIssues(ctx, user.ID)
	dailySummary, _ := h.Store.GetDailySummary(ctx, user.ID, time.Now().UTC())
	session, _ := h.Store.GetActiveWorkSession(ctx, user.ID)
	sessions, _ := h.Store.ListRecentWorkSessions(ctx, user.ID, 10)

	var sessionSecs int64
	if session != nil {
		sessionSecs = int64(time.Since(session.StartedAt).Seconds())
	}

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

	rows, err := h.Store.DB().QueryContext(ctx,
		`SELECT te.issue_id, te.started_at, te.duration, te.actor_type, te.mcp_actor_id,
		        i.number, i.title, i.project_id, p.key
		 FROM time_entries te
		 JOIN issues i ON te.issue_id = i.id
		 JOIN projects p ON i.project_id = p.id
		 WHERE te.user_id = ? AND te.started_at >= ? AND te.started_at < ? AND te.duration IS NOT NULL
		 ORDER BY i.project_id, i.number, te.actor_type, te.mcp_actor_id`,
		user.ID, weekStart, weekEnd)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tsMap := make(map[string]*timesheetEntry)
	var dailyTotal [7]int64
	var weekTotal int64

	for rows.Next() {
		var issueID, pKey, actorType string
		var mcp sql.NullString
		var startedAt time.Time
		var duration int64
		var num int
		var title, projectID string
		if err := rows.Scan(&issueID, &startedAt, &duration, &actorType, &mcp,
			&num, &title, &projectID, &pKey); err != nil {
			continue
		}
		dayIdx := int(startedAt.Weekday()) - 1
		if dayIdx < 0 {
			dayIdx = 6
		}

		actorSlot := ""
		if mcp.Valid {
			actorSlot = mcp.String
		}
		key := fmt.Sprintf("%s/%s/%s", issueID, actorType, actorSlot)
		e, ok := tsMap[key]
		if !ok {
			e = &timesheetEntry{
				IssueID: issueID, ProjectKey: pKey, Number: num, Title: title,
				ActorType: actorType, McpActorID: actorSlot,
			}
			tsMap[key] = e
		}
		e.Days[dayIdx] += duration
		e.Total += duration
		dailyTotal[dayIdx] += duration
		weekTotal += duration
	}

	tsEntries := make([]*timesheetEntry, 0, len(tsMap))
	for _, e := range tsMap {
		tsEntries = append(tsEntries, e)
	}

	pd := h.newPageData(r, "Time", timeHubData{
		ActiveTimers: activeTimers,
		DailySummary: dailySummary,
		Session:      session,
		SessionSecs:  sessionSecs,
		Sessions:     sessions,
		WeekStart:    weekStart,
		WeekEnd:      weekEnd,
		PrevWeek:     weekStart.AddDate(0, 0, -7).Format("2006-01-02"),
		NextWeek:     weekStart.AddDate(0, 0, 7).Format("2006-01-02"),
		TSEntries:    tsEntries,
		DailyTotal:   dailyTotal,
		WeekTotal:    weekTotal,
		DayNames:     [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	})
	pd.ActiveNav = "time"
	h.renderApp(w, r, "time-hub", timeHubTpl, pd)
}

type agentTimerInfo struct {
	ProjectKey string
	Number     int
	Title      string
	Secs       int64
	McpActorID string
}

type sessionBannerData struct {
	Session      *model.WorkSession
	SessionSecs  int64
	HumanTimer   *model.TimeEntry
	HumanIssue   *model.Issue
	HumanKey     string
	HumanSecs    int64
	AgentTimers  []agentTimerInfo
	AgentSecs    int64
	DailySummary *store.DailySummary
}

var sessionBannerTpl = template.Must(template.New("session-banner").Funcs(funcMap).Parse(`
{{if .Session}}
<div class="bg-blue-50 border-b border-blue-200 px-4 py-2 flex items-center justify-between text-sm"
     x-data="{ sessSecs: {{.SessionSecs}}, humanSecs: {{.HumanSecs}}, agentSecs: {{.AgentSecs}} }"
     x-init="setInterval(() => { sessSecs++; {{if .HumanTimer}}humanSecs++;{{end}} }, 1000)">
    <div class="flex items-center gap-6">
        <div class="flex items-center gap-2">
            <span class="inline-block w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
            <span class="font-medium text-blue-900">Session</span>
            <span class="font-mono text-blue-700" x-text="Math.floor(sessSecs/3600) + 'h ' + Math.floor((sessSecs%3600)/60) + 'm'"></span>
        </div>
        <div class="h-4 w-px bg-blue-200"></div>
        <div class="flex items-center gap-2">
            <svg class="w-3.5 h-3.5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
            {{if .HumanTimer}}
            <a href="/projects/{{.HumanKey}}/issues/{{.HumanIssue.Number}}" class="font-mono text-xs text-blue-700 hover:underline">{{.HumanKey}}-{{.HumanIssue.Number}}</a>
            <span class="font-mono text-blue-600" x-text="Math.floor(humanSecs/3600) + 'h ' + Math.floor((humanSecs%3600)/60) + 'm ' + (humanSecs%60) + 's'"></span>
            <button hx-post="/time/stop" hx-target="#session-banner" hx-swap="innerHTML"
                    class="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs hover:bg-red-200">Stop</button>
            {{else}}
            <span class="text-blue-400">idle</span>
            {{end}}
        </div>
        {{if .AgentTimers}}
        <div class="h-4 w-px bg-blue-200"></div>
        <div class="flex items-center gap-2">
            <svg class="w-3.5 h-3.5 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
            {{range .AgentTimers}}
            <a href="/projects/{{.ProjectKey}}/issues/{{.Number}}" class="text-purple-700 text-xs font-medium hover:underline">{{.ProjectKey}}-{{.Number}}</a>
            {{if .McpActorID}}<span class="text-slate-400 font-mono text-[10px]">{{.McpActorID}}</span>{{end}}
            {{end}}
            <span class="font-mono text-purple-600 text-xs" x-text="Math.floor(agentSecs/3600) + 'h ' + Math.floor((agentSecs%3600)/60) + 'm'"></span>
        </div>
        {{end}}
        {{if .DailySummary}}
        <div class="h-4 w-px bg-blue-200"></div>
        <div class="text-xs text-blue-500">
            Today: {{formatDuration .DailySummary.TotalSecs}} ({{.DailySummary.IssueCount}} issues)
        </div>
        {{end}}
    </div>
    <button hx-post="/session/end" hx-target="#session-banner" hx-swap="innerHTML"
            class="px-3 py-1 bg-blue-600 text-white rounded text-xs hover:bg-blue-700 font-medium">Clock Out</button>
</div>
{{else}}
<div class="bg-slate-50 border-b border-slate-200 px-4 py-2 flex items-center justify-between text-sm">
    <div class="flex items-center gap-3">
        <span class="inline-block w-2 h-2 rounded-full bg-slate-300"></span>
        <span class="text-slate-500">No active session</span>
        {{if .DailySummary}}
        <span class="text-xs text-slate-400 ml-2">Today: {{formatDuration .DailySummary.TotalSecs}}</span>
        {{end}}
    </div>
    <button hx-post="/session/start" hx-target="#session-banner" hx-swap="innerHTML"
            class="px-3 py-1 bg-emerald-600 text-white rounded text-xs hover:bg-emerald-700 font-medium">Clock In</button>
</div>
{{end}}`))

func (h *Handler) renderSessionBanner(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	ctx := r.Context()

	data := sessionBannerData{}
	data.Session, _ = h.Store.GetActiveWorkSession(ctx, user.ID)
	if data.Session != nil {
		data.SessionSecs = int64(time.Since(data.Session.StartedAt).Seconds())
	}

	data.HumanTimer, _ = h.Store.GetActiveTimer(ctx, user.ID)
	if data.HumanTimer != nil {
		data.HumanSecs = int64(time.Since(data.HumanTimer.StartedAt).Seconds())
		if issue, err := h.Store.GetIssue(ctx, data.HumanTimer.IssueID); err == nil {
			data.HumanIssue = issue
			if p, err := h.Store.GetProjectByID(ctx, issue.ProjectID); err == nil {
				data.HumanKey = p.Key
			}
		}
	}

	allTimers, _ := h.Store.GetActiveTimersWithIssues(ctx, user.ID)
	for _, t := range allTimers {
		if t.ActorType == "agent" {
			secs := int64(time.Since(t.StartedAt).Seconds())
			info := agentTimerInfo{Secs: secs, McpActorID: t.McpActorID}
			if t.Issue != nil {
				info.ProjectKey = t.Issue.ProjectKey
				info.Number = t.Issue.Number
				info.Title = t.Issue.Title
			}
			data.AgentTimers = append(data.AgentTimers, info)
			data.AgentSecs += secs
		}
	}

	data.DailySummary, _ = h.Store.GetDailySummary(ctx, user.ID, time.Now().UTC())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	sessionBannerTpl.Execute(w, data)
}

func (h *Handler) renderTimeEntries(w http.ResponseWriter, entries []model.TimeEntry) {
	tpl := template.Must(template.New("time-entries").Funcs(template.FuncMap{
		"formatDuration": FormatDuration,
		"derefInt64": func(i *int64) int64 {
			if i != nil {
				return *i
			}
			return 0
		},
	}).Parse(`
	{{range .}}
	<div class="flex items-center justify-between py-2 border-b border-slate-100 text-sm">
		<div class="flex items-center gap-3">
			{{if eq .ActorType "agent"}}<span class="text-[10px] font-mono text-purple-600">{{if .McpActorID}}{{.McpActorID}}{{else}}agent{{end}}</span>{{end}}
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

var timeHubTpl = template.Must(template.New("time-hub").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$d := .Content}}
<h1 class="text-xl font-semibold text-slate-900 mb-6">Time</h1>

<!-- Active Timers -->
<div class="bg-white rounded-lg border border-slate-200 p-4 mb-6">
    <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Active Timers</h2>
    {{if $d.ActiveTimers}}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {{range $d.ActiveTimers}}
        <div class="border border-slate-200 rounded-lg p-3"
             x-data="{ elapsed: {{elapsed .StartedAt}} }" x-init="setInterval(() => elapsed++, 1000)">
            <div class="flex items-center justify-between mb-1">
                <div class="flex items-center gap-2">
                    {{if eq .ActorType "agent"}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700">AI</span>
                    {{if .McpActorID}}<span class="font-mono text-[10px] text-slate-400 truncate max-w-[5rem]" title="{{.McpActorID}}">{{.McpActorID}}</span>{{end}}
                    {{else}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">You</span>
                    {{end}}
                    <a href="/projects/{{.Issue.ProjectKey}}/issues/{{.Issue.Number}}" class="font-mono text-sm font-medium text-blue-600 hover:underline">
                        {{.Issue.ProjectKey}}-{{.Issue.Number}}
                    </a>
                </div>
                <div class="flex items-center gap-2">
                    <span class="font-mono text-sm text-slate-700" x-text="Math.floor(elapsed/3600) + 'h ' + Math.floor((elapsed%3600)/60) + 'm ' + (elapsed%60) + 's'"></span>
                    {{if eq .ActorType "human"}}
                    <button hx-post="/time/stop" hx-target="#session-banner" hx-swap="innerHTML"
                            class="px-2 py-1 bg-red-50 text-red-600 rounded text-xs hover:bg-red-100"
                            onclick="setTimeout(() => location.reload(), 300)">Stop</button>
                    {{end}}
                </div>
            </div>
            <div class="text-sm text-slate-600 truncate pl-7">{{.Issue.Title}}</div>
        </div>
        {{end}}
    </div>
    {{else}}
    <p class="text-sm text-slate-400">No active timers.</p>
    {{end}}
</div>

<!-- Today Summary -->
<div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <div class="text-xs font-medium text-slate-500 uppercase tracking-wide mb-1">Today Total</div>
        <div class="text-2xl font-bold text-slate-900">{{formatDuration $d.DailySummary.TotalSecs}}</div>
    </div>
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <div class="text-xs font-medium text-slate-500 uppercase tracking-wide mb-1">Human Time</div>
        <div class="text-2xl font-bold text-blue-600">{{formatDuration $d.DailySummary.HumanSecs}}</div>
    </div>
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <div class="text-xs font-medium text-slate-500 uppercase tracking-wide mb-1">Agent Time</div>
        <div class="text-2xl font-bold text-purple-600">{{formatDuration $d.DailySummary.AgentSecs}}</div>
    </div>
    <div class="bg-white rounded-lg border border-slate-200 p-4">
        <div class="text-xs font-medium text-slate-500 uppercase tracking-wide mb-1">Issues Worked</div>
        <div class="text-2xl font-bold text-slate-900">{{$d.DailySummary.IssueCount}}</div>
    </div>
</div>

<!-- Session History -->
<div class="bg-white rounded-lg border border-slate-200 p-4 mb-6">
    <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide">Session History</h2>
        {{if $d.Session}}
        <button hx-post="/session/end" hx-target="#session-banner" hx-swap="innerHTML"
                class="px-3 py-1 bg-blue-600 text-white rounded text-xs hover:bg-blue-700 font-medium"
                onclick="setTimeout(() => location.reload(), 300)">Clock Out</button>
        {{else}}
        <button hx-post="/session/start" hx-target="#session-banner" hx-swap="innerHTML"
                class="px-3 py-1 bg-emerald-600 text-white rounded text-xs hover:bg-emerald-700 font-medium"
                onclick="setTimeout(() => location.reload(), 300)">Clock In</button>
        {{end}}
    </div>
    {{if $d.Sessions}}
    <table class="w-full text-sm">
        <thead class="border-b border-slate-200">
            <tr>
                <th class="text-left px-3 py-2 font-medium text-slate-500">Date</th>
                <th class="text-left px-3 py-2 font-medium text-slate-500">Description</th>
                <th class="text-right px-3 py-2 font-medium text-slate-500">Duration</th>
                <th class="text-right px-3 py-2 font-medium text-slate-500">Status</th>
            </tr>
        </thead>
        <tbody>
            {{range $d.Sessions}}
            <tr class="border-b border-slate-100">
                <td class="px-3 py-2 text-slate-500">{{.StartedAt.Format "Jan 2, 3:04 PM"}}</td>
                <td class="px-3 py-2">{{if .Description}}{{.Description}}{{else}}<span class="text-slate-400">-</span>{{end}}</td>
                <td class="px-3 py-2 text-right font-mono">
                    {{if .Duration}}{{formatDuration (derefInt64 .Duration)}}{{else}}
                    <span class="text-emerald-600 font-medium">active</span>
                    {{end}}
                </td>
                <td class="px-3 py-2 text-right">
                    {{if .EndedAt}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-slate-100 text-slate-500">ended</span>
                    {{else}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-emerald-100 text-emerald-700">active</span>
                    {{end}}
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p class="text-sm text-slate-400">No sessions recorded yet.</p>
    {{end}}
</div>

<!-- Weekly Timesheet -->
<div class="bg-white rounded-lg border border-slate-200 p-4">
    <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Weekly Timesheet</h2>
    <div class="flex items-center gap-4 mb-4">
        <a href="/time?week={{$d.PrevWeek}}" class="px-3 py-1 border border-slate-300 rounded hover:bg-slate-100 text-sm">&lt; Prev</a>
        <span class="font-medium text-sm">{{$d.WeekStart.Format "Jan 2"}} - {{$d.WeekEnd.Format "Jan 2, 2006"}}</span>
        <a href="/time?week={{$d.NextWeek}}" class="px-3 py-1 border border-slate-300 rounded hover:bg-slate-100 text-sm">Next &gt;</a>
    </div>
    <div class="overflow-x-auto">
    <table class="w-full text-sm">
        <thead class="bg-slate-50 border-b border-slate-200">
            <tr>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Issue</th>
                <th class="text-left px-2 py-2 font-medium text-slate-500 w-10">Type</th>
                {{range $d.DayNames}}
                <th class="text-center px-3 py-2 font-medium text-slate-500 w-16">{{.}}</th>
                {{end}}
                <th class="text-center px-3 py-2 font-medium text-slate-500 w-16">Total</th>
            </tr>
        </thead>
        <tbody>
            {{range $d.TSEntries}}
            <tr class="border-b border-slate-100">
                <td class="px-4 py-2">
                    <a href="/projects/{{.ProjectKey}}/issues/{{.Number}}" class="hover:text-blue-600">
                        <div class="font-mono text-xs text-slate-500">{{.ProjectKey}}-{{.Number}}</div>
                        <div class="text-sm truncate max-w-xs">{{.Title}}</div>
                    </a>
                </td>
                <td class="px-2 py-2">
                    {{if eq .ActorType "agent"}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700">AI</span>
                    {{if .McpActorID}}<div class="font-mono text-[10px] text-slate-400 truncate max-w-[4rem]" title="{{.McpActorID}}">{{.McpActorID}}</div>{{end}}
                    {{else}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">You</span>
                    {{end}}
                </td>
                {{range .Days}}
                <td class="text-center px-3 py-2 text-xs">
                    {{if gt . 0}}{{formatDuration .}}{{else}}<span class="text-slate-300">-</span>{{end}}
                </td>
                {{end}}
                <td class="text-center px-3 py-2 text-xs font-medium">{{formatDuration .Total}}</td>
            </tr>
            {{end}}
            {{if not $d.TSEntries}}
            <tr><td colspan="10" class="px-4 py-8 text-center text-slate-400">No time tracked this week.</td></tr>
            {{end}}
        </tbody>
        <tfoot class="bg-slate-50 border-t border-slate-200">
            <tr>
                <td class="px-4 py-2 font-medium" colspan="2">Daily Total</td>
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
</div>
{{end}}`))
