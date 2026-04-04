package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
)

func (h *Handler) PostTag(w http.ResponseWriter, r *http.Request) {
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

	tag := r.FormValue("tag")
	groupName := r.FormValue("group_name")
	if tag == "" {
		http.Error(w, "Tag required", http.StatusBadRequest)
		return
	}

	h.Store.AddProjectTag(r.Context(), project.ID, tag, groupName)

	// Return updated tag list
	h.renderProjectTags(w, r, project.ID, key)
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	key := h.urlParam(r, "key")
	tag := h.urlParam(r, "tag")
	project, err := h.Store.GetProjectByKey(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	h.Store.RemoveProjectTag(r.Context(), project.ID, tag)
	h.renderProjectTags(w, r, project.ID, key)
}

func (h *Handler) SuggestTags(w http.ResponseWriter, r *http.Request) {
	q := h.queryParam(r, "q")
	tags, _ := h.Store.SuggestTags(r.Context(), q)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *Handler) renderProjectTags(w http.ResponseWriter, r *http.Request, projectID, key string) {
	tags, _ := h.Store.GetProjectTags(r.Context(), projectID)

	tagsTpl := template.Must(template.New("tags").Parse(`
	<div class="flex flex-wrap gap-2 mb-3" id="project-tags">
		{{range .Tags}}
		<span class="inline-flex items-center gap-1 px-2 py-1 bg-slate-100 rounded text-xs">
			{{.Tag}}
			{{if .GroupName}}<span class="text-slate-400">{{.GroupName}}</span>{{end}}
			<button hx-delete="/projects/{{$.Key}}/tags/{{.Tag}}" hx-target="#project-tags" hx-swap="outerHTML"
					class="text-slate-400 hover:text-red-600">&times;</button>
		</span>
		{{end}}
	</div>`))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tagsTpl.Execute(w, struct {
		Tags interface{}
		Key  string
	}{Tags: tags, Key: key})
}
