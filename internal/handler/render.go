package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"yaitracker.com/loweryaustin/internal/middleware"
	"yaitracker.com/loweryaustin/internal/model"
)

var funcMap = template.FuncMap{
	"formatDuration": FormatDuration,
	"timeAgo":        TimeAgo,
	"elapsed":        func(t time.Time) int64 { return int64(time.Since(t).Seconds()) },
	"join":           strings.Join,
	"contains":       contains,
	"statusColor":    StatusColor,
	"priorityColor":  PriorityColor,
	"deref":          func(i *int) int { if i != nil { return *i }; return 0 },
	"derefStr":       func(s *string) string { if s != nil { return *s }; return "" },
	"derefFloat":     func(f *float64) float64 { if f != nil { return *f }; return 0 },
	"derefInt64":     func(i *int64) int64 { if i != nil { return *i }; return 0 },
	"statuses":       func() []string { return model.IssueStatuses },
	"types":          func() []string { return model.IssueTypes },
	"priorities":     func() []string { return model.IssuePriorities },
	"budgetPct": func(totalSecs int64, estHours float64) float64 {
		if estHours <= 0 {
			return 0
		}
		return float64(totalSecs) / (estHours * 3600) * 100
	},
	"subtract":       func(a, b int) int { return a - b },
	"add":            func(a, b int) int { return a + b },
	"hasMore":        func(total, page, perPage int) bool { return page*perPage < total },
	"min":            func(a, b int) int { if a < b { return a }; return b },
	"mul":            func(a, b int) int { return a * b },
	// gt in text/template requires identical types; use gtI/gtF to avoid float64/int mismatches.
	"gtI":            func(a, b int) bool { return a > b },
	"gtF":            func(a, b float64) bool { return a > b },
	"statusDot": func(status string) string {
		switch status {
		case "backlog":
			return "bg-slate-400"
		case "todo":
			return "bg-blue-400"
		case "in_progress":
			return "bg-amber-500"
		case "in_review":
			return "bg-purple-500"
		case "done":
			return "bg-emerald-500"
		case "cancelled":
			return "bg-slate-300"
		default:
			return "bg-slate-400"
		}
	},
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

func FormatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func TimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func StatusColor(status string) string {
	switch status {
	case "backlog":
		return "bg-slate-100 text-slate-600"
	case "todo":
		return "bg-blue-100 text-blue-700"
	case "in_progress":
		return "bg-amber-100 text-amber-700"
	case "in_review":
		return "bg-purple-100 text-purple-700"
	case "done":
		return "bg-emerald-100 text-emerald-700"
	case "cancelled":
		return "bg-slate-100 text-slate-400 line-through"
	default:
		return "bg-slate-100 text-slate-600"
	}
}

func PriorityColor(priority string) string {
	switch priority {
	case "urgent":
		return "text-red-600"
	case "high":
		return "text-orange-500"
	case "medium":
		return "text-yellow-500"
	case "low":
		return "text-blue-400"
	default:
		return "text-slate-400"
	}
}

type pageData struct {
	Title      string
	User       *model.User
	CSRFToken  string
	Nonce      string
	Content    interface{}
	Error      string
	ActiveNav  string
	ProjectKey string
	ActiveTab  string
}

func (h *Handler) newPageData(r *http.Request, title string, content interface{}) pageData {
	return pageData{
		Title:     title,
		User:      h.currentUser(r),
		CSRFToken: middleware.GetCSRFToken(r),
		Nonce:     middleware.GetCSPNonce(r.Context()),
		Content:   content,
	}
}

const baseTpl = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - YAITracker</title>
    <script nonce="{{.Nonce}}" src="/static/js/htmx.min.js"></script>
    <script nonce="{{.Nonce}}" defer src="/static/js/alpine.min.js"></script>
    <script nonce="{{.Nonce}}" src="/static/js/sortable.min.js"></script>
    <link rel="stylesheet" href="/static/css/app.css">
    <script nonce="{{.Nonce}}" src="/static/js/app.js" defer></script>
</head>
<body class="bg-slate-50 text-slate-700 text-sm" hx-headers='{"X-CSRF-Token": "{{.CSRFToken}}"}'>`

const authLayout = baseTpl + `
<div class="min-h-screen flex items-center justify-center">
    <div class="w-full max-w-md">
        <div class="text-center mb-8">
            <h1 class="text-2xl font-bold text-slate-900">YAITracker</h1>
        </div>
        <div class="bg-white rounded-lg shadow-sm border border-slate-200 p-8">
            {{template "content" .}}
        </div>
    </div>
</div>
</body></html>`

const appLayout = baseTpl + `
<div class="min-h-screen flex flex-col">
    <!-- Top Bar -->
    <header class="h-16 bg-white border-b border-slate-200 flex items-center px-4 sticky top-0 z-30">
        <a href="/dashboard" class="font-bold text-lg text-slate-900 mr-8">YAITracker</a>
        <div class="flex-1 max-w-xl mx-auto">
            <input type="search" placeholder="Search issues... (Cmd+K)"
                   class="w-full px-3 py-1.5 bg-slate-100 rounded-md border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                   hx-get="/search" hx-trigger="keyup changed delay:200ms" hx-target="#search-results" name="q">
            <div id="search-results"></div>
        </div>
        <div class="ml-4 flex items-center gap-2" x-data="{ open: false }">
            <button @click="open = !open" class="flex items-center gap-2 px-2 py-1 rounded hover:bg-slate-100">
                <div class="w-7 h-7 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-medium">
                    {{if .User}}{{slice .User.Name 0 1}}{{end}}
                </div>
                <span class="text-sm">{{if .User}}{{.User.Name}}{{end}}</span>
            </button>
            <div x-show="open" @click.outside="open = false" x-cloak
                 class="absolute right-4 top-14 bg-white rounded-md shadow-lg border border-slate-200 py-1 w-48 z-50">
                <form method="POST" action="/logout">
                    <input type="hidden" name="_csrf" value="{{.CSRFToken}}">
                    <button type="submit" class="w-full text-left px-4 py-2 hover:bg-slate-100">Log out</button>
                </form>
            </div>
        </div>
    </header>
    <div id="session-banner" hx-get="/partials/session-banner" hx-trigger="load" hx-swap="innerHTML"></div>
    <div class="flex flex-1">
        <!-- Sidebar -->
        <aside class="w-60 bg-white border-r border-slate-200 p-4 flex-shrink-0 overflow-y-auto" id="sidebar">
            <nav class="space-y-1">
                <a href="/dashboard"
                   class="flex items-center gap-2 px-3 py-2 rounded-md {{if eq .ActiveNav "dashboard"}}bg-blue-50 text-blue-700 font-medium{{else}}hover:bg-slate-100{{end}}">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/></svg>
                    Dashboard
                </a>
            </nav>
            <div class="mt-6">
                <div class="flex items-center justify-between px-3 mb-2">
                    <span class="text-xs font-medium text-slate-500 uppercase tracking-wide">Projects</span>
                    <a href="/projects/new" class="text-blue-600 hover:text-blue-800 text-xs font-medium">+ New</a>
                </div>
                <div id="project-nav" hx-get="/partials/project-nav" hx-trigger="load" hx-swap="innerHTML">
                </div>
            </div>
            <div class="mt-6 space-y-1">
                <a href="/time"
                   class="flex items-center gap-2 px-3 py-2 rounded-md {{if eq .ActiveNav "time"}}bg-blue-50 text-blue-700 font-medium{{else}}hover:bg-slate-100{{end}}">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                    Time
                </a>
                <a href="/analytics/compare"
                   class="flex items-center gap-2 px-3 py-2 rounded-md {{if eq .ActiveNav "compare"}}bg-blue-50 text-blue-700 font-medium{{else}}hover:bg-slate-100{{end}}">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>
                    Compare
                </a>
            </div>
        </aside>
        <!-- Main content -->
        <main class="flex-1 p-6 overflow-auto">
            {{template "content" .}}
        </main>
    </div>
</div>
<!-- Toast container -->
<div id="toast" class="fixed top-4 right-4 z-50" x-data="{ toasts: [] }">
</div>
</body></html>`

var (
	loginTpl    *template.Template
	registerTpl *template.Template
)

func init() {
	loginTpl = template.Must(template.New("login").Funcs(funcMap).Parse(authLayout + `
{{define "content"}}
<h2 class="text-lg font-semibold text-slate-900 mb-6">Log in to YAITracker</h2>
{{if .Error}}<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">{{.Error}}</div>{{end}}
<form method="POST" action="/login" class="space-y-4">
    <input type="hidden" name="_csrf" value="{{.CSRFToken}}">
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Email</label>
        <input type="email" name="email" value="{{if .Content}}{{.Content}}{{end}}" required autofocus
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Password</label>
        <input type="password" name="password" required
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <button type="submit" class="w-full py-2 px-4 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium">
        Log in
    </button>
</form>
{{end}}`))

	registerTpl = template.Must(template.New("register").Funcs(funcMap).Parse(authLayout + `
{{define "content"}}
<h2 class="text-lg font-semibold text-slate-900 mb-6">Create your admin account</h2>
{{if .Error}}<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">{{.Error}}</div>{{end}}
<form method="POST" action="/register" class="space-y-4">
    <input type="hidden" name="_csrf" value="{{.CSRFToken}}">
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Name</label>
        <input type="text" name="name" required autofocus
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Email</label>
        <input type="email" name="email" required
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Password</label>
        <input type="password" name="password" required minlength="12"
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
        <p class="text-xs text-slate-500 mt-1">Minimum 12 characters</p>
    </div>
    <button type="submit" class="w-full py-2 px-4 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium">
        Create Account
    </button>
</form>
{{end}}`))
}

func (h *Handler) renderLogin(w http.ResponseWriter, r *http.Request, errorMsg, email string) {
	pd := h.newPageData(r, "Login", email)
	pd.Error = errorMsg
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTpl.Execute(w, pd)
}

func (h *Handler) renderRegister(w http.ResponseWriter, r *http.Request, errorMsg string, firstRun bool) {
	pd := h.newPageData(r, "Register", nil)
	pd.Error = errorMsg
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	registerTpl.Execute(w, pd)
}

func (h *Handler) renderApp(w http.ResponseWriter, r *http.Request, tplName string, tpl *template.Template, pd pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, pd); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}
