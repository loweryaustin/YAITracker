package handler

import (
	"html/template"
	"net/http"

	"yaitracker.com/loweryaustin/internal/model"
)

func (h *Handler) PostComment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
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
	body := r.FormValue("body")
	if body == "" {
		http.Error(w, "Comment body required", http.StatusBadRequest)
		return
	}

	comment := &model.Comment{
		IssueID:  issue.ID,
		AuthorID: user.ID,
		Body:     body,
	}

	if err := h.Store.CreateComment(r.Context(), comment); err != nil {
		http.Error(w, "Could not create comment", http.StatusInternalServerError)
		return
	}

	h.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit log
		EntityType: "issue", EntityID: issue.ID, UserID: user.ID,
		Action: "commented", IPAddress: h.clientIP(r),
	})

	// Return the comment HTML fragment for htmx append
	commentFragment := template.Must(template.New("comment-frag").Funcs(funcMap).Parse(`
	<div class="flex gap-3" id="comment-{{.ID}}">
		<div class="w-7 h-7 rounded-full bg-slate-300 flex items-center justify-center text-xs text-white font-medium flex-shrink-0">
			{{slice .Author.Name 0 1}}
		</div>
		<div class="flex-1">
			<div class="flex items-center gap-2 mb-1">
				<span class="font-medium text-sm">{{.Author.Name}}</span>
				<span class="text-xs text-slate-400">just now</span>
			</div>
			<div class="text-sm text-slate-700 prose prose-sm max-w-none">{{.Body | markdown}}</div>
		</div>
	</div>`))

	comment.Author = user
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := commentFragment.Execute(w, comment); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) PatchComment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	id := h.urlParam(r, "id")
	comment, err := h.Store.GetComment(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	user := h.currentUser(r)
	if comment.AuthorID != user.ID && !user.IsAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	comment.Body = r.FormValue("body")
	if err := h.Store.UpdateComment(r.Context(), comment); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id := h.urlParam(r, "id")
	comment, err := h.Store.GetComment(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user := h.currentUser(r)
	if comment.AuthorID != user.ID && !user.IsAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Store.DeleteComment(r.Context(), id); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
