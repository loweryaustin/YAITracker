package handler

import (
	"html/template"
	"net/http"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
)

type dashboardData struct {
	Projects     []model.ProjectSummary
	Activities   []model.ActivityLog
	DailySummary *store.DailySummary
	Session      *model.WorkSession
	SessionSecs  int64
	ActiveCount  int
}

var dashboardTpl = template.Must(template.New("dashboard").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
<h1 class="text-xl font-semibold text-slate-900 mb-6">Dashboard</h1>

<!-- Today's Time Card -->
<div class="bg-white rounded-lg border border-slate-200 p-4 mb-6">
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-6">
            <div>
                <div class="text-xs font-medium text-slate-500 uppercase tracking-wide">Session</div>
                {{if .Content.Session}}
                <div class="text-sm font-medium text-emerald-600"
                     x-data="{ s: {{.Content.SessionSecs}} }" x-init="setInterval(() => s++, 1000)"
                     x-text="'Active ' + Math.floor(s/3600) + 'h ' + Math.floor((s%3600)/60) + 'm'">
                </div>
                {{else}}
                <div class="text-sm text-slate-400">Not clocked in</div>
                {{end}}
            </div>
            <div class="h-8 w-px bg-slate-200"></div>
            <div>
                <div class="text-xs font-medium text-slate-500 uppercase tracking-wide">Today</div>
                <div class="text-sm font-mono font-medium">{{formatDuration .Content.DailySummary.TotalSecs}}</div>
            </div>
            <div class="h-8 w-px bg-slate-200"></div>
            <div>
                <div class="text-xs font-medium text-slate-500 uppercase tracking-wide">Human / Agent</div>
                <div class="text-sm">
                    <span class="font-mono text-blue-600">{{formatDuration .Content.DailySummary.HumanSecs}}</span>
                    <span class="text-slate-300 mx-1">/</span>
                    <span class="font-mono text-purple-600">{{formatDuration .Content.DailySummary.AgentSecs}}</span>
                </div>
            </div>
            {{if gt .Content.ActiveCount 0}}
            <div class="h-8 w-px bg-slate-200"></div>
            <div>
                <div class="text-xs font-medium text-slate-500 uppercase tracking-wide">Active Timers</div>
                <div class="text-sm font-medium">{{.Content.ActiveCount}}</div>
            </div>
            {{end}}
        </div>
        <a href="/time" class="text-xs text-blue-600 hover:underline">View all &rarr;</a>
    </div>
</div>

<div class="grid grid-cols-3 gap-6">
    <div class="col-span-2 space-y-4">
        <div class="flex items-center justify-between mb-2">
            <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide">Your Projects</h2>
        </div>
        {{if not .Content.Projects}}
        <div class="text-center py-12 bg-white rounded-lg border border-slate-200">
            <p class="text-slate-500 mb-4">No projects yet. Create your first project to get started.</p>
            <a href="/projects/new" class="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">+ New Project</a>
        </div>
        {{else}}
        {{range .Content.Projects}}
        <a href="/projects/{{.Key}}/board" class="block bg-white rounded-lg border border-slate-200 p-4 hover:border-slate-300 transition-colors">
            <div class="flex items-center justify-between mb-2">
                <div>
                    <span class="font-mono text-xs text-slate-500">{{.Key}}</span>
                    <h3 class="font-medium text-slate-900">{{.Name}}</h3>
                </div>
                <span class="text-xs text-slate-500">{{.OpenIssues}} open, {{.InProgressIssues}} in progress</span>
            </div>
            <div class="w-full bg-slate-100 rounded-full h-1.5 mb-2">
                <div class="bg-emerald-500 h-1.5 rounded-full" style="width: {{printf "%.0f" .ProgressPercent}}%"></div>
            </div>
            <div class="flex items-center gap-1">
                {{range .Tags}}
                <span class="px-1.5 py-0.5 bg-slate-100 text-slate-500 rounded text-xs">{{.Tag}}</span>
                {{end}}
            </div>
        </a>
        {{end}}
        {{end}}
    </div>
    <div>
        <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Recent Activity</h2>
        <div class="space-y-3" id="activity-feed">
            {{range .Content.Activities}}
            <div class="text-sm">
                <div class="flex items-center gap-2">
                    <div class="w-6 h-6 rounded-full bg-slate-300 flex items-center justify-center text-xs text-white font-medium">
                        {{if .User}}{{slice .User.Name 0 1}}{{end}}
                    </div>
                    <div>
                        <span class="font-medium">{{if .User}}{{.User.Name}}{{end}}</span>
                        <span class="text-slate-500">{{.Action}}</span>
                        {{if .Field}}<span class="text-slate-500">{{.Field}}</span>{{end}}
                    </div>
                </div>
                <div class="ml-8 text-xs text-slate-400">{{timeAgo .CreatedAt}}</div>
            </div>
            {{end}}
            {{if not .Content.Activities}}
            <p class="text-sm text-slate-400">No activity yet.</p>
            {{end}}
        </div>
    </div>
</div>
{{end}}`))

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	ctx := r.Context()
	projects, _ := h.Store.ListProjectSummaries(ctx)
	activities, _ := h.Store.ListRecentActivity(ctx, 20)
	dailySummary, _ := h.Store.GetDailySummary(ctx, user.ID, time.Now().UTC())
	session, _ := h.Store.GetActiveWorkSession(ctx, user.ID)
	activeTimers, _ := h.Store.GetActiveTimers(ctx, user.ID)

	var sessionSecs int64
	if session != nil {
		sessionSecs = int64(time.Since(session.StartedAt).Seconds())
	}

	pd := h.newPageData(r, "Dashboard", dashboardData{
		Projects:     projects,
		Activities:   activities,
		DailySummary: dailySummary,
		Session:      session,
		SessionSecs:  sessionSecs,
		ActiveCount:  len(activeTimers),
	})
	pd.ActiveNav = "dashboard"
	h.renderApp(w, r, "dashboard", dashboardTpl, pd)
}

func (h *Handler) GetProjectNav(w http.ResponseWriter, r *http.Request) {
	projects, _ := h.Store.ListProjects(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	navTpl := template.Must(template.New("nav").Parse(`
	{{range .}}
	<div class="mb-1" x-data="{ expanded: false }">
		<button @click="expanded = !expanded"
				class="w-full flex items-center gap-2 px-3 py-1.5 rounded-md hover:bg-slate-100 text-left">
			<svg class="w-3 h-3 transition-transform" :class="expanded ? 'rotate-90' : ''" fill="currentColor" viewBox="0 0 20 20"><path d="M6 6l4 4-4 4V6z"/></svg>
			<span class="font-mono text-xs text-slate-500">{{.Key}}</span>
			<span class="truncate">{{.Name}}</span>
		</button>
		<div x-show="expanded" x-cloak class="ml-8 space-y-0.5 mt-0.5">
			<a href="/projects/{{.Key}}/board" class="block px-3 py-1 rounded-md hover:bg-slate-100 text-sm">Board</a>
			<a href="/projects/{{.Key}}/issues" class="block px-3 py-1 rounded-md hover:bg-slate-100 text-sm">Issues</a>
			<a href="/projects/{{.Key}}/analytics" class="block px-3 py-1 rounded-md hover:bg-slate-100 text-sm">Analytics</a>
			<a href="/projects/{{.Key}}/settings" class="block px-3 py-1 rounded-md hover:bg-slate-100 text-sm">Settings</a>
		</div>
	</div>
	{{end}}`))
	navTpl.Execute(w, projects)
}
