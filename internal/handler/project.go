package handler

import (
	"html/template"
	"net/http"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

var newProjectTpl = template.Must(template.New("new-project").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
<div class="max-w-2xl">
<h1 class="text-xl font-semibold text-slate-900 mb-6">New Project</h1>
{{if .Error}}<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">{{.Error}}</div>{{end}}
<form method="POST" action="/projects" class="space-y-4 bg-white rounded-lg border border-slate-200 p-6">
    <input type="hidden" name="_csrf" value="{{.CSRFToken}}">
    <div class="grid grid-cols-2 gap-4">
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Project Key *</label>
            <input type="text" name="key" required maxlength="10" pattern="[A-Z]+" placeholder="TRACK"
                   class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono uppercase">
            <p class="text-xs text-slate-500 mt-1">Uppercase letters only, used in issue numbers</p>
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Name *</label>
            <input type="text" name="name" required placeholder="YAITracker"
                   class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
        </div>
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Description</label>
        <textarea name="description" rows="3" placeholder="What is this project about?"
                  class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"></textarea>
    </div>
    <div class="grid grid-cols-2 gap-4">
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Target Date</label>
            <input type="date" name="target_date"
                   class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Budget Hours</label>
            <input type="number" name="budget_hours" step="0.5" min="0" placeholder="80"
                   class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
        </div>
    </div>
    <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Tags</label>
        <input type="text" name="tags" placeholder="go, web, htmx (comma separated)"
               class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
    </div>
    <div class="flex items-center gap-3 pt-2">
        <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">Create Project</button>
        <a href="/dashboard" class="px-4 py-2 text-slate-600 hover:text-slate-800 text-sm">Cancel</a>
    </div>
</form>
</div>
{{end}}`))

func (h *Handler) GetNewProject(w http.ResponseWriter, r *http.Request) {
	pd := h.newPageData(r, "New Project", nil)
	h.renderApp(w, r, "new-project", newProjectTpl, pd)
}

func (h *Handler) PostProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		pd := h.newPageData(r, "New Project", nil)
		pd.Error = "Invalid form data"
		h.renderApp(w, r, "new-project", newProjectTpl, pd)
		return
	}

	user := h.currentUser(r)
	key := strings.ToUpper(strings.TrimSpace(r.FormValue("key")))
	name := strings.TrimSpace(r.FormValue("name"))

	if key == "" || name == "" {
		pd := h.newPageData(r, "New Project", nil)
		pd.Error = "Key and name are required"
		h.renderApp(w, r, "new-project", newProjectTpl, pd)
		return
	}

	p := &model.Project{
		Key:         key,
		Name:        name,
		Description: r.FormValue("description"),
		Status:      "active",
		CreatedBy:   user.ID,
	}

	if td := r.FormValue("target_date"); td != "" {
		t, err := parseDate(td)
		if err == nil {
			p.TargetDate = &t
		}
	}
	if bh := r.FormValue("budget_hours"); bh != "" {
		if f, err := parseFloat(bh); err == nil {
			p.BudgetHours = &f
		}
	}

	if err := h.Store.CreateProject(r.Context(), p); err != nil {
		pd := h.newPageData(r, "New Project", nil)
		pd.Error = "Could not create project: " + err.Error()
		h.renderApp(w, r, "new-project", newProjectTpl, pd)
		return
	}

	// Add tags
	if tagsStr := r.FormValue("tags"); tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if tag != "" {
				h.Store.AddProjectTag(r.Context(), p.ID, tag, "")
			}
		}
	}

	h.Store.LogActivity(r.Context(), &model.ActivityLog{
		EntityType: "project", EntityID: p.ID, UserID: user.ID,
		Action: "created", IPAddress: h.clientIP(r),
	})

	http.Redirect(w, r, "/projects/"+p.Key+"/board", http.StatusSeeOther)
}

func (h *Handler) GetProjectSettings(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	members, _ := h.Store.GetProjectMembers(r.Context(), project.ID)
	labels, _ := h.Store.ListLabels(r.Context(), project.ID)

	type settingsData struct {
		Project *model.Project
		Members []model.ProjectMember
		Labels  []model.Label
	}

	pd := h.newPageData(r, project.Name+" Settings", settingsData{
		Project: project,
		Members: members,
		Labels:  labels,
	})
	pd.ProjectKey = key
	pd.ActiveTab = "settings"
	h.renderApp(w, r, "project-settings", projectSettingsTpl, pd)
}

func (h *Handler) PostProjectSettings(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	project.Name = r.FormValue("name")
	project.Description = r.FormValue("description")
	project.Status = r.FormValue("status")

	if td := r.FormValue("target_date"); td != "" {
		t, err := parseDate(td)
		if err == nil {
			project.TargetDate = &t
		}
	} else {
		project.TargetDate = nil
	}

	if bh := r.FormValue("budget_hours"); bh != "" {
		if f, err := parseFloat(bh); err == nil {
			project.BudgetHours = &f
		}
	} else {
		project.BudgetHours = nil
	}

	h.Store.UpdateProject(r.Context(), project)

	user := h.currentUser(r)
	h.Store.LogActivity(r.Context(), &model.ActivityLog{
		EntityType: "project", EntityID: project.ID, UserID: user.ID,
		Action: "updated_settings", IPAddress: h.clientIP(r),
	})

	http.Redirect(w, r, "/projects/"+key+"/settings", http.StatusSeeOther)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user := h.currentUser(r)
	h.Store.LogActivity(r.Context(), &model.ActivityLog{
		EntityType: "project", EntityID: project.ID, UserID: user.ID,
		Action: "deleted_project", NewValue: project.Name, IPAddress: h.clientIP(r),
	})

	if err := h.Store.DeleteProject(r.Context(), project.ID); err != nil {
		http.Error(w, "Could not delete project", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

var projectSettingsTpl = template.Must(template.New("project-settings").Funcs(funcMap).Parse(appLayout + `
{{define "content"}}
{{$p := .Content.Project}}
<div class="mb-4">
    <span class="font-mono text-xs text-slate-500">{{$p.Key}}</span>
    <span class="mx-1 text-slate-300">&gt;</span>
    <span>Settings</span>
</div>
<div class="flex gap-2 border-b border-slate-200 mb-6">
    <a href="/projects/{{$p.Key}}/board" class="px-4 py-2 text-sm hover:text-blue-600">Board</a>
    <a href="/projects/{{$p.Key}}/issues" class="px-4 py-2 text-sm hover:text-blue-600">Issues</a>
    <a href="/projects/{{$p.Key}}/analytics" class="px-4 py-2 text-sm hover:text-blue-600">Analytics</a>
    <a href="/projects/{{$p.Key}}/settings" class="px-4 py-2 text-sm border-b-2 border-blue-600 text-blue-600 font-medium">Settings</a>
</div>
<div class="max-w-2xl space-y-8">
    <form method="POST" action="/projects/{{$p.Key}}/settings" class="bg-white rounded-lg border border-slate-200 p-6 space-y-4">
        <input type="hidden" name="_csrf" value="{{$.CSRFToken}}">
        <h2 class="font-medium text-slate-900">General</h2>
        <div class="grid grid-cols-2 gap-4">
            <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Key</label>
                <input type="text" value="{{$p.Key}}" disabled class="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-md font-mono text-sm">
            </div>
            <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Name</label>
                <input type="text" name="name" value="{{$p.Name}}" required class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
            </div>
        </div>
        <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Description</label>
            <textarea name="description" rows="3" class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">{{$p.Description}}</textarea>
        </div>
        <div class="grid grid-cols-3 gap-4">
            <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Status</label>
                <select name="status" class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                    <option value="active" {{if eq $p.Status "active"}}selected{{end}}>Active</option>
                    <option value="paused" {{if eq $p.Status "paused"}}selected{{end}}>Paused</option>
                    <option value="completed" {{if eq $p.Status "completed"}}selected{{end}}>Completed</option>
                    <option value="archived" {{if eq $p.Status "archived"}}selected{{end}}>Archived</option>
                </select>
            </div>
            <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Target Date</label>
                <input type="date" name="target_date" {{if $p.TargetDate}}value="{{$p.TargetDate.Format "2006-01-02"}}"{{end}}
                       class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
            </div>
            <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Budget Hours</label>
                <input type="number" name="budget_hours" step="0.5" {{if $p.BudgetHours}}value="{{printf "%.1f" (index (slice (printf "%v" $p.BudgetHours) 1) 0)}}"{{end}}
                       class="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
            </div>
        </div>
        <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm">Save</button>
    </form>

    <div class="bg-white rounded-lg border border-slate-200 p-6">
        <h2 class="font-medium text-slate-900 mb-4">Tags</h2>
        <div class="flex flex-wrap gap-2 mb-3" id="project-tags">
            {{range $p.Tags}}
            <span class="inline-flex items-center gap-1 px-2 py-1 bg-slate-100 rounded text-xs">
                {{.Tag}}
                {{if .GroupName}}<span class="text-slate-400">{{.GroupName}}</span>{{end}}
                <button hx-delete="/projects/{{$p.Key}}/tags/{{.Tag}}" hx-target="#project-tags" hx-swap="outerHTML"
                        class="text-slate-400 hover:text-red-600">&times;</button>
            </span>
            {{end}}
        </div>
        <form hx-post="/projects/{{$p.Key}}/tags" hx-target="#project-tags" hx-swap="outerHTML" class="flex gap-2">
            <input type="text" name="tag" placeholder="Tag name" required class="px-3 py-1.5 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            <input type="text" name="group_name" placeholder="Group (optional)" class="px-3 py-1.5 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            <button type="submit" class="px-3 py-1.5 bg-slate-100 rounded-md hover:bg-slate-200 text-sm">Add</button>
        </form>
    </div>

    <div class="bg-white rounded-lg border border-red-200 p-6">
        <h2 class="font-medium text-red-700 mb-2">Danger Zone</h2>
        <p class="text-sm text-slate-600 mb-4">Deleting a project removes all issues, comments, and time entries permanently.</p>
        <button hx-delete="/projects/{{$p.Key}}" hx-confirm="Are you sure you want to delete this project? This cannot be undone."
                class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 font-medium text-sm">Delete Project</button>
    </div>
</div>
{{end}}`))
