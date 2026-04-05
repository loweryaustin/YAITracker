package api

import (
	"net/http"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

// PostMCPActor registers a new MCP actor for the authenticated user.
func (a *API) PostMCPActor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label string `json:"label"`
	}
	if err := a.decodeJSON(r, &req); err != nil {
		a.jsonError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	user := a.currentUser(r)
	actor, err := a.Store.CreateMCPActor(r.Context(), user.ID, strings.TrimSpace(req.Label))
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not register MCP actor")
		return
	}
	a.jsonCreated(w, actor)
}

// ListMCPActors lists registered MCP actors for the authenticated user.
func (a *API) ListMCPActors(w http.ResponseWriter, r *http.Request) {
	user := a.currentUser(r)
	actors, err := a.Store.ListMCPActors(r.Context(), user.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "server_error", "Could not list MCP actors")
		return
	}
	if actors == nil {
		actors = []model.MCPActor{}
	}
	a.jsonOK(w, actors)
}

// PostMCPActorHeartbeat updates the heartbeat timestamp for an actor.
func (a *API) PostMCPActorHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := a.currentUser(r)
	id := a.urlParam(r, "id")
	if err := a.Store.TouchActorHeartbeat(r.Context(), user.ID, id); err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMCPActor revokes an MCP actor by id.
func (a *API) DeleteMCPActor(w http.ResponseWriter, r *http.Request) {
	user := a.currentUser(r)
	id := a.urlParam(r, "id")
	if err := a.Store.RevokeMCPActor(r.Context(), user.ID, id); err != nil {
		a.jsonError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
