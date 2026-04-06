package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

type issueListData struct {
	Project *model.Project
	Issues  []model.Issue
	Total   int
	Filter  model.IssueFilter
	Page    int
	PerPage int
}

var issueListTpl = template.Must(template.New("issue-list").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$p := .Content.Project}}
<div class="mb-4">
    <span class="font-mono text-xs text-slate-500">{{$p.Key}}</span>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span>Issues</span>
</div>
<div class="flex gap-2 border-b border-slate-200 mb-6">
    <a href="/projects/{{$p.Key}}/board" class="px-4 py-2 text-sm hover:text-blue-600">Board</a>
    <a href="/projects/{{$p.Key}}/issues" class="px-4 py-2 text-sm border-b-2 border-blue-600 text-blue-600 font-medium">Issues</a>
    <a href="/projects/{{$p.Key}}/analytics" class="px-4 py-2 text-sm hover:text-blue-600">Analytics</a>
    <a href="/projects/{{$p.Key}}/settings" class="px-4 py-2 text-sm hover:text-blue-600">Settings</a>
</div>
<div class="flex items-center justify-between mb-4">
    <a href="/projects/{{$p.Key}}/issues/new" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">+ New Issue</a>
    <form class="flex items-center gap-2" method="GET">
        <input type="search" name="q" value="{{.Content.Filter.Query}}" placeholder="Search..."
               class="px-3 py-1.5 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
    </form>
</div>
<div class="bg-white rounded-lg border border-slate-200 overflow-hidden">
    <table class="w-full text-sm">
        <thead class="bg-slate-50 border-b border-slate-200">
            <tr>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Key</th>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Title</th>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Status</th>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Priority</th>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Assignee</th>
                <th class="text-left px-4 py-2 font-medium text-slate-500">Pts</th>
            </tr>
        </thead>
        <tbody id="issue-table-body">
            {{range .Content.Issues}}
            <tr class="border-b border-slate-100 hover:bg-slate-50 cursor-pointer" onclick="window.location='/projects/{{$p.Key}}/issues/{{.Number}}'">
                <td class="px-4 py-2 font-mono text-xs text-slate-500">{{$p.Key}}-{{.Number}}</td>
                <td class="px-4 py-2">{{.Title}}</td>
                <td class="px-4 py-2"><span class="px-2 py-0.5 rounded text-xs font-medium {{statusColor .Status}}">{{.Status}}</span></td>
                <td class="px-4 py-2"><span class="text-xs {{priorityColor .Priority}}">{{.Priority}}</span></td>
                <td class="px-4 py-2 text-xs text-slate-500">{{if .Assignee}}{{.Assignee.Name}}{{else}}-{{end}}</td>
                <td class="px-4 py-2 text-xs">{{if .StoryPoints}}{{deref .StoryPoints}}{{else}}-{{end}}</td>
            </tr>
            {{end}}
            {{if not .Content.Issues}}
            <tr><td colspan="6" class="px-4 py-8 text-center text-slate-400">No issues match your filters.</td></tr>
            {{end}}
        </tbody>
    </table>
</div>
<div class="flex items-center justify-between mt-4 text-sm text-slate-500">
    <span>Showing {{len .Content.Issues}} of {{.Content.Total}} issues</span>
    <div class="flex gap-2">
        {{if gt .Content.Page 1}}
        <a href="?page={{subtract .Content.Page 1}}" class="px-3 py-1 border border-slate-300 rounded hover:bg-slate-100">&lt; Prev</a>
        {{end}}
        {{if hasMore .Content.Total .Content.Page .Content.PerPage}}
        <a href="?page={{add .Content.Page 1}}" class="px-3 py-1 border border-slate-300 rounded hover:bg-slate-100">Next &gt;</a>
        {{end}}
    </div>
</div>
{{end}}`))

func (h *Handler) GetIssueList(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	page := h.queryParamInt(r, "page", "1")
	if page < 1 {
		page = 1
	}
	perPage := 25

	filter := model.IssueFilter{
		ProjectID: project.ID,
		Query:     h.queryParam(r, "q"),
		SortBy:    h.queryParam(r, "sort"),
		SortDir:   h.queryParam(r, "dir"),
		Limit:     perPage,
		Offset:    (page - 1) * perPage,
	}

	if statuses := h.queryParam(r, "status"); statuses != "" {
		filter.Status = strings.Split(statuses, ",")
	}
	if types := h.queryParam(r, "type"); types != "" {
		filter.Type = strings.Split(types, ",")
	}

	issues, total, _ := h.Store.ListIssues(r.Context(), filter)

	// Load assignees
	for i := range issues {
		if issues[i].AssigneeID != nil {
			issues[i].Assignee, _ = h.Store.GetUserByID(r.Context(), *issues[i].AssigneeID)
		}
	}

	pd := h.newPageData(r, project.Name+" - Issues", issueListData{
		Project: project,
		Issues:  issues,
		Total:   total,
		Filter:  filter,
		Page:    page,
		PerPage: perPage,
	})
	pd.ProjectKey = key
	pd.ActiveTab = "issues"
	h.renderApp(w, r, "issue-list", issueListTpl, pd)
}

type issueDetailData struct {
	Project     *model.Project
	Issue       *model.Issue
	Comments    []model.Comment
	Activities  []model.ActivityLog
	TimeEntries []model.TimeEntry
	TotalTime   int64
	HumanTime   int64
	AgentTime   int64
	Children    []model.Issue
	Labels      []model.Label
	AllLabels   []model.Label
	Users       []model.User
	HasActive   bool
}

var issueDetailTpl = template.Must(template.New("issue-detail").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$p := .Content.Project}}
{{$i := .Content.Issue}}
<div class="mb-4 text-sm">
    <a href="/projects/{{$p.Key}}/issues" class="text-blue-600 hover:underline">{{$p.Key}}</a>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span>Issues</span>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span class="font-mono">{{$p.Key}}-{{$i.Number}}</span>
</div>
<div class="grid grid-cols-3 gap-6">
    <!-- Main content -->
    <div class="col-span-2 space-y-6">
        <div>
            <div class="flex items-center gap-2 mb-1">
                <span class="text-xs {{priorityColor $i.Priority}}">●</span>
                <span class="font-mono text-xs text-slate-500">{{$p.Key}}-{{$i.Number}}</span>
            </div>
            <h1 class="text-xl font-semibold text-slate-900" id="issue-title">{{$i.Title}}</h1>
        </div>

        <!-- Description -->
        <div class="bg-white rounded-lg border border-slate-200 p-4">
            <div class="flex items-center justify-between mb-2">
                <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide">Description</h2>
            </div>
            <div class="prose prose-sm max-w-none">
                {{if $i.Description}}{{$i.Description | markdown}}{{else}}<span class="text-slate-400">No description.</span>{{end}}
            </div>
        </div>

        <!-- Time log -->
        <div class="bg-white rounded-lg border border-slate-200 p-4">
            <div class="flex items-center justify-between mb-3">
                <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide">Time Tracking</h2>
                {{if .Content.HasActive}}
                <span class="text-xs px-2 py-1 bg-emerald-100 text-emerald-700 rounded font-medium">Timer Running</span>
                {{else}}
                <button hx-post="/time/start" hx-vals='{"issue_id":"{{$i.ID}}"}' hx-target="#session-banner" hx-swap="innerHTML"
                        class="text-xs px-2 py-1 bg-emerald-50 text-emerald-700 rounded hover:bg-emerald-100 font-medium">▶ Start Timer</button>
                {{end}}
            </div>
            {{if $i.EstimatedHours}}
            <div class="mb-3">
                <div class="flex items-center justify-between text-xs text-slate-500 mb-1">
                    <span>{{formatDuration .Content.TotalTime}} logged</span>
                    <span>{{printf "%.0f" (derefFloat $i.EstimatedHours)}}h estimated</span>
                </div>
                {{$pct := budgetPct .Content.TotalTime (derefFloat $i.EstimatedHours)}}
                <div class="w-full bg-slate-100 rounded-full h-2">
                    <div class="h-2 rounded-full {{if lt $pct 80.0}}bg-emerald-500{{else if le $pct 100.0}}bg-amber-500{{else}}bg-red-500{{end}}"
                         style="width: {{if gt $pct 100.0}}100{{else}}{{printf "%.0f" $pct}}{{end}}%"></div>
                </div>
            </div>
            {{end}}
            <div class="flex gap-4 mb-3">
                <div class="flex items-center gap-2 text-sm">
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">Human</span>
                    <span class="font-mono">{{formatDuration .Content.HumanTime}}</span>
                </div>
                <div class="flex items-center gap-2 text-sm">
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700">Agent</span>
                    <span class="font-mono">{{formatDuration .Content.AgentTime}}</span>
                </div>
                <div class="text-sm font-medium ml-auto">
                    Total: {{formatDuration .Content.TotalTime}}
                </div>
            </div>
            <div id="time-entries">
            {{range .Content.TimeEntries}}
            <div class="flex items-center justify-between py-2 border-b border-slate-100 text-sm">
                <div class="flex items-center gap-2">
                    {{if eq .ActorType "agent"}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700">AI</span>
                    {{if .McpActorID}}<span class="font-mono text-[10px] text-slate-400 max-w-[8rem] truncate" title="{{.McpActorID}}">{{.McpActorID}}</span>{{end}}
                    {{else}}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">You</span>
                    {{end}}
                    <span>{{if .User}}{{.User.Name}}{{end}}</span>
                    <span class="text-slate-400">{{.StartedAt.Format "Jan 2"}}</span>
                    {{if eq .Source "manual"}}
                    <span class="text-slate-300 text-xs">(manual)</span>
                    {{end}}
                </div>
                <div class="flex items-center gap-3">
                    <span class="font-mono">{{if .Duration}}{{formatDuration (derefInt64 .Duration)}}{{else}}<span class="text-emerald-600">running...</span>{{end}}</span>
                    {{if .Description}}<span class="text-slate-400 text-xs">{{.Description}}</span>{{end}}
                </div>
            </div>
            {{end}}
            {{if not .Content.TimeEntries}}
            <p class="py-4 text-center text-slate-400 text-sm">No time logged yet.</p>
            {{end}}
            </div>
        </div>

        <!-- Comments -->
        <div class="bg-white rounded-lg border border-slate-200 p-4">
            <h2 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-3">Activity & Comments</h2>
            <div id="comment-thread" class="space-y-4">
                {{range .Content.Comments}}
                <div class="flex gap-3" id="comment-{{.ID}}">
                    <div class="w-7 h-7 rounded-full bg-slate-300 flex items-center justify-center text-xs text-white font-medium flex-shrink-0">
                        {{if .Author}}{{slice .Author.Name 0 1}}{{end}}
                    </div>
                    <div class="flex-1">
                        <div class="flex items-center gap-2 mb-1">
                            <span class="font-medium text-sm">{{if .Author}}{{.Author.Name}}{{end}}</span>
                            <span class="text-xs text-slate-400">{{timeAgo .CreatedAt}}</span>
                        </div>
                        <div class="text-sm text-slate-700 prose prose-sm max-w-none">{{.Body | markdown}}</div>
                    </div>
                </div>
                {{end}}
                {{range .Content.Activities}}
                <div class="flex gap-3 text-xs text-slate-400">
                    <div class="w-7 h-7 flex items-center justify-center flex-shrink-0">·</div>
                    <div>
                        <span class="font-medium text-slate-500">{{if .User}}{{.User.Name}}{{end}}</span>
                        {{.Action}} {{.Field}}
                        {{if .OldValue}}<span class="line-through">{{.OldValue}}</span> →{{end}}
                        {{if .NewValue}}{{.NewValue}}{{end}}
                        <span class="ml-1">{{timeAgo .CreatedAt}}</span>
                    </div>
                </div>
                {{end}}
            </div>
            <form hx-post="/projects/{{$p.Key}}/issues/{{$i.Number}}/comments" hx-target="#comment-thread" hx-swap="beforeend"
                  class="mt-4 pt-4 border-t border-slate-100">
                <textarea name="body" rows="3" placeholder="Add a comment..." required
                          class="w-full px-3 py-2 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 mb-2"></textarea>
                <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">Post Comment</button>
            </form>
        </div>
    </div>

    <!-- Sidebar -->
    <div class="space-y-4">
        <div class="bg-white rounded-lg border border-slate-200 p-4 space-y-4 sticky top-20">
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Status</label>
                <select hx-patch="/projects/{{$p.Key}}/issues/{{$i.Number}}" name="status" hx-target="body"
                        class="w-full px-2 py-1 border border-slate-200 rounded text-sm">
                    {{range statuses}}
                    <option value="{{.}}" {{if eq . $i.Status}}selected{{end}}>{{.}}</option>
                    {{end}}
                </select>
            </div>
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Priority</label>
                <select hx-patch="/projects/{{$p.Key}}/issues/{{$i.Number}}" name="priority" hx-target="body"
                        class="w-full px-2 py-1 border border-slate-200 rounded text-sm">
                    {{range priorities}}
                    <option value="{{.}}" {{if eq . $i.Priority}}selected{{end}}>{{.}}</option>
                    {{end}}
                </select>
            </div>
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Type</label>
                <select hx-patch="/projects/{{$p.Key}}/issues/{{$i.Number}}" name="type" hx-target="body"
                        class="w-full px-2 py-1 border border-slate-200 rounded text-sm">
                    {{range types}}
                    <option value="{{.}}" {{if eq . $i.Type}}selected{{end}}>{{.}}</option>
                    {{end}}
                </select>
            </div>
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Assignee</label>
                <select hx-patch="/projects/{{$p.Key}}/issues/{{$i.Number}}" name="assignee_id" hx-target="body"
                        class="w-full px-2 py-1 border border-slate-200 rounded text-sm">
                    <option value="">Unassigned</option>
                    {{range $.Content.Users}}
                    <option value="{{.ID}}" {{if and $i.AssigneeID (eq .ID (derefStr $i.AssigneeID))}}selected{{end}}>{{.Name}}</option>
                    {{end}}
                </select>
            </div>
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Labels</label>
                <div class="flex flex-wrap gap-1">
                    {{range $i.Labels}}
                    <span class="px-2 py-0.5 rounded text-xs" style="background-color: {{.Color}}20; color: {{.Color}}">{{.Name}}</span>
                    {{end}}
                </div>
            </div>
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Estimate</label>
                <div class="flex gap-2 text-sm">
                    <span>{{if $i.StoryPoints}}{{deref $i.StoryPoints}} pts{{else}}-{{end}}</span>
                    <span class="text-slate-300">|</span>
                    <span>{{if $i.EstimatedHours}}{{printf "%.1f" (derefFloat $i.EstimatedHours)}}h{{else}}-{{end}}</span>
                </div>
            </div>
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Dates</label>
                <div class="text-xs text-slate-500 space-y-1">
                    <div>Created: {{$i.CreatedAt.Format "Jan 2, 2006"}}</div>
                    {{if $i.StartedAt}}<div>Started: {{$i.StartedAt.Format "Jan 2, 2006"}}</div>{{end}}
                    {{if $i.CompletedAt}}<div>Completed: {{$i.CompletedAt.Format "Jan 2, 2006"}}</div>{{end}}
                </div>
            </div>
            {{if .Content.Children}}
            <div>
                <label class="text-xs font-medium text-slate-500 uppercase tracking-wide block mb-1">Children ({{len .Content.Children}})</label>
                <div class="space-y-1">
                    {{range .Content.Children}}
                    <a href="/projects/{{$p.Key}}/issues/{{.Number}}" class="flex items-center gap-2 text-sm hover:text-blue-600">
                        <span class="font-mono text-xs text-slate-500">{{$p.Key}}-{{.Number}}</span>
                        <span class="px-1.5 py-0.5 rounded text-xs {{statusColor .Status}}">{{.Status}}</span>
                    </a>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
    </div>
</div>
{{end}}`))

func (h *Handler) GetIssueDetail(w http.ResponseWriter, r *http.Request) {
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

	if issue.AssigneeID != nil {
		issue.Assignee, _ = h.Store.GetUserByID(r.Context(), *issue.AssigneeID)
	}

	comments, _ := h.Store.ListComments(r.Context(), issue.ID)
	activities, _ := h.Store.ListActivity(r.Context(), "issue", issue.ID, 50, 0)
	timeEntries, _ := h.Store.ListTimeEntries(r.Context(), issue.ID)
	totalTime, _ := h.Store.GetIssueTotalTime(r.Context(), issue.ID)
	children, _ := h.Store.GetChildIssues(r.Context(), issue.ID)
	labels, _ := h.Store.ListLabels(r.Context(), project.ID)
	users, _ := h.Store.ListUsers(r.Context())

	var humanTime, agentTime int64
	var hasActive bool
	for _, te := range timeEntries {
		if te.Duration != nil {
			if te.ActorType == "agent" {
				agentTime += *te.Duration
			} else {
				humanTime += *te.Duration
			}
		}
		if te.EndedAt == nil {
			hasActive = true
		}
	}

	pd := h.newPageData(r, fmt.Sprintf("%s-%d: %s", project.Key, issue.Number, issue.Title), issueDetailData{
		Project:     project,
		Issue:       issue,
		Comments:    comments,
		Activities:  activities,
		TimeEntries: timeEntries,
		TotalTime:   totalTime,
		HumanTime:   humanTime,
		AgentTime:   agentTime,
		Children:    children,
		Labels:      issue.Labels,
		AllLabels:   labels,
		Users:       users,
		HasActive:   hasActive,
	})
	pd.ProjectKey = key
	h.renderApp(w, r, "issue-detail", issueDetailTpl, pd)
}

var newIssueTpl = template.Must(template.New("new-issue").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$p := .Content.Project}}
<div class="mb-4 text-sm">
    <a href="/projects/{{$p.Key}}/issues" class="text-blue-600 hover:underline">{{$p.Key}}</a>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span>New Issue</span>
</div>
<div class="max-w-2xl">
<h1 class="text-xl font-semibold text-slate-900 mb-6">New Issue</h1>
{{if .Error}}<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">{{.Error}}</div>{{end}}
<form method="POST" action="/projects/{{$p.Key}}/issues" class="space-y-4 bg-white rounded-lg border border-slate-200 p-6">
    <input type="hidden" name="_csrf" value="{{$.CSRFToken}}">
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Title *</label>
        <input type="text" name="title" required autofocus
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Description</label>
        <textarea name="description" rows="6" placeholder="Markdown supported..."
                  class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"></textarea>
    </div>
    <div class="grid grid-cols-3 gap-4">
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Type</label>
            <select name="type" class="w-full px-3 py-2 border border-slate-300 rounded-md">
                <option value="task">Task</option>
                <option value="bug">Bug</option>
                <option value="feature">Feature</option>
                <option value="improvement">Improvement</option>
            </select>
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Status</label>
            <select name="status" class="w-full px-3 py-2 border border-slate-300 rounded-md">
                <option value="backlog">Backlog</option>
                <option value="todo">Todo</option>
            </select>
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Priority</label>
            <select name="priority" class="w-full px-3 py-2 border border-slate-300 rounded-md">
                <option value="none">None</option>
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="urgent">Urgent</option>
            </select>
        </div>
    </div>
    <div class="grid grid-cols-3 gap-4">
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Assignee</label>
            <select name="assignee_id" class="w-full px-3 py-2 border border-slate-300 rounded-md">
                <option value="">Unassigned</option>
                {{range .Content.Users}}
                <option value="{{.ID}}">{{.Name}}</option>
                {{end}}
            </select>
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Story Points</label>
            <input type="number" name="story_points" min="0" class="w-full px-3 py-2 border border-slate-300 rounded-md">
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Est. Hours</label>
            <input type="number" name="estimated_hours" min="0" step="0.5" class="w-full px-3 py-2 border border-slate-300 rounded-md">
        </div>
    </div>
    <div class="flex items-center gap-3 pt-2">
        <button type="submit" name="action" value="create" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">Create Issue</button>
        <button type="submit" name="action" value="create_new" class="px-4 py-2 bg-slate-100 text-slate-700 rounded-md hover:bg-slate-200 text-sm">Create & New</button>
        <a href="/projects/{{$p.Key}}/issues" class="px-4 py-2 text-slate-600 hover:text-slate-800 text-sm">Cancel</a>
    </div>
</form>
</div>
{{end}}`))

type newIssueData struct {
	Project *model.Project
	Users   []model.User
}

func (h *Handler) GetNewIssue(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	users, _ := h.Store.ListUsers(r.Context())
	pd := h.newPageData(r, "New Issue", newIssueData{Project: project, Users: users})
	pd.ProjectKey = key
	h.renderApp(w, r, "new-issue", newIssueTpl, pd)
}

func (h *Handler) PostIssue(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	user := h.currentUser(r)
	issue := &model.Issue{
		ProjectID:   project.ID,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Type:        r.FormValue("type"),
		Status:      r.FormValue("status"),
		Priority:    r.FormValue("priority"),
		ReporterID:  user.ID,
	}

	if aid := r.FormValue("assignee_id"); aid != "" {
		issue.AssigneeID = &aid
	}
	if sp := r.FormValue("story_points"); sp != "" {
		if v, err := strconv.Atoi(sp); err == nil {
			issue.StoryPoints = &v
		}
	}
	if eh := r.FormValue("estimated_hours"); eh != "" {
		if v, err := parseFloat(eh); err == nil {
			issue.EstimatedHours = &v
		}
	}

	if issue.Title == "" {
		users, _ := h.Store.ListUsers(r.Context())
		pd := h.newPageData(r, "New Issue", newIssueData{Project: project, Users: users})
		pd.Error = "Title is required"
		h.renderApp(w, r, "new-issue", newIssueTpl, pd)
		return
	}

	if err := h.Store.CreateIssue(r.Context(), issue); err != nil {
		users, _ := h.Store.ListUsers(r.Context())
		pd := h.newPageData(r, "New Issue", newIssueData{Project: project, Users: users})
		pd.Error = "Could not create issue: " + err.Error()
		h.renderApp(w, r, "new-issue", newIssueTpl, pd)
		return
	}

	h.Store.LogActivity(r.Context(), &model.ActivityLog{
		EntityType: "issue", EntityID: issue.ID, UserID: user.ID,
		Action: "created", IPAddress: h.clientIP(r),
	})

	action := r.FormValue("action")
	if action == "create_new" {
		http.Redirect(w, r, "/projects/"+key+"/issues/new", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/projects/%s/issues/%d", key, issue.Number), http.StatusSeeOther)
	}
}

func (h *Handler) PatchIssue(w http.ResponseWriter, r *http.Request) {
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
	changed := false

	if v := r.FormValue("title"); v != "" && v != issue.Title {
		h.logFieldChange(r, issue.ID, user.ID, "title", issue.Title, v)
		issue.Title = v
		changed = true
	}
	if r.FormValue("description") != "" {
		issue.Description = r.FormValue("description")
		changed = true
	}
	if v := r.FormValue("status"); v != "" && v != issue.Status {
		h.logFieldChange(r, issue.ID, user.ID, "status", issue.Status, v)
		h.Store.UpdateIssueStatus(r.Context(), issue.ID, v)
		http.Redirect(w, r, fmt.Sprintf("/projects/%s/issues/%d", key, number), http.StatusSeeOther)
		return
	}
	if v := r.FormValue("priority"); v != "" && v != issue.Priority {
		h.logFieldChange(r, issue.ID, user.ID, "priority", issue.Priority, v)
		issue.Priority = v
		changed = true
	}
	if v := r.FormValue("type"); v != "" && v != issue.Type {
		h.logFieldChange(r, issue.ID, user.ID, "type", issue.Type, v)
		issue.Type = v
		changed = true
	}
	if r.Form.Has("assignee_id") {
		v := r.FormValue("assignee_id")
		if v == "" {
			issue.AssigneeID = nil
		} else {
			issue.AssigneeID = &v
		}
		changed = true
	}

	if changed {
		h.Store.UpdateIssue(r.Context(), issue)
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%s/issues/%d", key, number), http.StatusSeeOther)
}

func (h *Handler) DeleteIssue(w http.ResponseWriter, r *http.Request) {
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

	user := h.currentUser(r)
	h.Store.LogActivity(r.Context(), &model.ActivityLog{
		EntityType: "issue", EntityID: issue.ID, UserID: user.ID,
		Action: "deleted", NewValue: issue.Title, IPAddress: h.clientIP(r),
	})

	h.Store.DeleteIssue(r.Context(), issue.ID)
	http.Redirect(w, r, "/projects/"+key+"/issues", http.StatusSeeOther)
}

func (h *Handler) logFieldChange(r *http.Request, issueID, userID, field, oldVal, newVal string) {
	h.Store.LogActivity(r.Context(), &model.ActivityLog{
		EntityType: "issue", EntityID: issueID, UserID: userID,
		Action: "changed", Field: field, OldValue: oldVal, NewValue: newVal,
		IPAddress: h.clientIP(r),
	})
}
