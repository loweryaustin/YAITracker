package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"yaitracker.com/loweryaustin/internal/model"
)

type boardData struct {
	Project  *model.Project
	Columns  map[string][]model.Issue
	Statuses []string
}

var boardTpl = template.Must(template.New("board").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$p := .Content.Project}}
<div class="mb-4">
    <span class="font-mono text-xs text-slate-500">{{$p.Key}}</span>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span>Board</span>
</div>
<div class="flex gap-2 border-b border-slate-200 mb-6">
    <a href="/projects/{{$p.Key}}/board" class="px-4 py-2 text-sm border-b-2 border-blue-600 text-blue-600 font-medium">Board</a>
    <a href="/projects/{{$p.Key}}/issues" class="px-4 py-2 text-sm hover:text-blue-600">Issues</a>
    <a href="/projects/{{$p.Key}}/analytics" class="px-4 py-2 text-sm hover:text-blue-600">Analytics</a>
    <a href="/projects/{{$p.Key}}/settings" class="px-4 py-2 text-sm hover:text-blue-600">Settings</a>
</div>
<div class="flex gap-4 overflow-x-auto pb-4" id="board">
    {{range .Content.Statuses}}
    {{$issues := index $.Content.Columns .}}
    <div class="flex-shrink-0 w-72">
        <div class="flex items-center justify-between mb-3 px-1">
            <h3 class="text-sm font-medium text-slate-700 flex items-center gap-2">
                <span class="w-2 h-2 rounded-full {{statusDot .}}"></span>
                {{.}} <span class="text-slate-400 font-normal">({{len $issues}})</span>
            </h3>
        </div>
        <div class="space-y-2 min-h-[100px] p-1 rounded-lg border-2 border-dashed border-transparent"
             id="column-{{.}}"
             data-status="{{.}}">
            {{range $issues}}
            <a href="/projects/{{$p.Key}}/issues/{{.Number}}"
               class="block bg-white rounded-lg border border-slate-200 p-3 hover:border-slate-300 shadow-sm cursor-pointer"
               data-issue-id="{{.ID}}" draggable="true">
                <div class="flex items-center gap-1.5 mb-1">
                    <span class="font-mono text-xs text-slate-400">{{$p.Key}}-{{.Number}}</span>
                </div>
                <div class="text-sm font-medium text-slate-800 mb-2 line-clamp-2">{{.Title}}</div>
                <div class="flex items-center justify-between">
                    <span class="text-xs {{priorityColor .Priority}}">{{.Priority}}</span>
                    <div class="flex items-center gap-1">
                        {{if .StoryPoints}}<span class="text-xs text-slate-400">{{deref .StoryPoints}}pts</span>{{end}}
                    </div>
                </div>
            </a>
            {{end}}
            {{if not $issues}}
            <div class="text-center py-8 text-xs text-slate-400">No issues</div>
            {{end}}
        </div>
        <!-- Quick add -->
        <div class="mt-2" x-data="{ open: false }">
            <button @click="open = true" x-show="!open" class="w-full text-left px-3 py-2 text-sm text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100">+ Add issue</button>
            <form x-show="open" x-cloak @keydown.escape="open = false"
                  hx-post="/projects/{{$p.Key}}/issues" hx-target="#column-{{.}}" hx-swap="beforeend"
                  class="bg-white rounded-lg border border-slate-200 p-2">
                <input type="hidden" name="status" value="{{.}}">
                <input type="hidden" name="_csrf" value="{{$.CSRFToken}}">
                <input type="text" name="title" placeholder="Issue title..." required autofocus
                       class="w-full px-2 py-1 text-sm border border-slate-200 rounded mb-1 focus:outline-none focus:ring-2 focus:ring-blue-500">
                <div class="flex gap-1">
                    <button type="submit" class="px-2 py-1 bg-blue-600 text-white rounded text-xs">Add</button>
                    <button type="button" @click="open = false" class="px-2 py-1 text-slate-400 text-xs">Cancel</button>
                </div>
            </form>
        </div>
    </div>
    {{end}}
</div>
<script nonce="{{$.Nonce}}">
document.querySelectorAll('[id^="column-"]').forEach(function(col) {
    new Sortable(col, {
        group: 'board',
        animation: 150,
        ghostClass: 'opacity-50',
        onEnd: function(evt) {
            var issueId = evt.item.dataset.issueId;
            var newStatus = evt.to.dataset.status;
            var items = evt.to.querySelectorAll('[data-issue-id]');
            var sortOrder = Array.from(items).indexOf(evt.item) * 1000;
            fetch('/projects/{{$p.Key}}/board/move', {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': document.querySelector('[name="_csrf"]')?.value || ''
                },
                body: JSON.stringify({issue_id: issueId, status: newStatus, sort_order: sortOrder})
            }).then(function() { htmx.ajax('GET', window.location.href, {target: '#board', swap: 'outerHTML'}); });
        }
    });
});
</script>
{{end}}`))

func (h *Handler) GetBoard(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	columns, err := h.Store.ListIssuesByStatus(r.Context(), project.ID)
	if err != nil {
		log.Printf("list issues by status: %v", err)
	}

	pd := h.newPageData(r, project.Name+" - Board", boardData{
		Project:  project,
		Columns:  columns,
		Statuses: model.IssueStatuses,
	})
	pd.ProjectKey = key
	pd.ActiveTab = "board"
	h.renderApp(w, "board", boardTpl, pd)
}

func (h *Handler) PatchBoardMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID   string  `json:"issue_id"`
		Status    string  `json:"status"`
		SortOrder float64 `json:"sort_order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user := h.currentUser(r)

	// Get current status for activity log
	issue, err := h.Store.GetIssue(r.Context(), req.IssueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	if issue.Status != req.Status {
		h.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit log
			EntityType: "issue", EntityID: issue.ID, UserID: user.ID,
			Action: "changed", Field: "status", OldValue: issue.Status, NewValue: req.Status,
			IPAddress: h.clientIP(r),
		})
	}

	if err := h.Store.MoveIssue(r.Context(), req.IssueID, req.Status, req.SortOrder); err != nil {
		http.Error(w, "move failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
