package server

import (
	"io/fs"
	"net/http"
	"time"

	yaitracker "yaitracker.com/loweryaustin"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	mcpgo "github.com/mark3labs/mcp-go/server"
	"yaitracker.com/loweryaustin/internal/api"
	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/handler"
	"yaitracker.com/loweryaustin/internal/middleware"
)

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	h := handler.New(s.store, s.secret)

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.BodyLimit(1 << 20))

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Embedded static assets
	staticSub, _ := fs.Sub(yaitracker.StaticFS, "static")
	fileServer := http.FileServer(http.FS(staticSub))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Auth pages (no session required)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRF)
		r.Use(middleware.RateLimitAuth)

		r.Get("/login", h.GetLogin)
		r.Post("/login", h.PostLogin)
		r.Get("/register", h.GetRegister)
		r.Post("/register", h.PostRegister)
	})

	r.Post("/logout", h.PostLogout)

	// Partials (authenticated, no CSRF needed for GET)
	r.Group(func(r chi.Router) {
		r.Use(auth.SessionMiddleware(s.store))
		r.Get("/partials/project-nav", h.GetProjectNav)
		r.Get("/partials/session-banner", h.GetSessionBanner)
	})

	// Authenticated HTML routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRF)
		r.Use(auth.SessionMiddleware(s.store))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		})
		r.Get("/dashboard", h.GetDashboard)

		// Projects
		r.Get("/projects/new", h.GetNewProject)
		r.Post("/projects", h.PostProject)
		r.Get("/projects/{key}/settings", h.GetProjectSettings)
		r.Post("/projects/{key}/settings", h.PostProjectSettings)
		r.Delete("/projects/{key}", h.DeleteProject)

		// Board
		r.Get("/projects/{key}/board", h.GetBoard)
		r.Patch("/projects/{key}/board/move", h.PatchBoardMove)

		// Issues
		r.Get("/projects/{key}/issues", h.GetIssueList)
		r.Get("/projects/{key}/issues/new", h.GetNewIssue)
		r.Post("/projects/{key}/issues", h.PostIssue)
		r.Get("/projects/{key}/issues/{number}", h.GetIssueDetail)
		r.Patch("/projects/{key}/issues/{number}", h.PatchIssue)
		r.Delete("/projects/{key}/issues/{number}", h.DeleteIssue)

		// Comments
		r.Post("/projects/{key}/issues/{number}/comments", h.PostComment)
		r.Patch("/comments/{id}", h.PatchComment)
		r.Delete("/comments/{id}", h.DeleteComment)

		// Labels
		r.Post("/projects/{key}/labels", h.PostLabel)
		r.Patch("/labels/{id}", h.PatchLabel)
		r.Delete("/labels/{id}", h.DeleteLabel)

		// Tags
		r.Post("/projects/{key}/tags", h.PostTag)
		r.Delete("/projects/{key}/tags/{tag}", h.DeleteTag)
		r.Get("/tags/suggest", h.SuggestTags)

		// Time tracking
		r.Post("/time/start", h.PostStartTimer)
		r.Post("/time/stop", h.PostStopTimer)
		r.Post("/projects/{key}/issues/{number}/time", h.PostManualTimeEntry)
		r.Patch("/time/{id}", h.PatchTimeEntry)
		r.Delete("/time/{id}", h.DeleteTimeEntry)
		r.Get("/time", h.GetTimeHub)
		r.Get("/time/sheet", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/time", http.StatusMovedPermanently)
		})
		r.Post("/session/start", h.PostSessionStart)
		r.Post("/session/end", h.PostSessionEnd)

		// Analytics
		r.Get("/projects/{key}/analytics", h.GetProjectAnalytics)
		r.Get("/analytics/compare", h.GetCompare)
		r.Get("/analytics/predict", h.GetPredict)
	})

	// JSON API
	a := api.New(s.store)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RateLimitAPI)
		if s.cors != "" {
			r.Use(middleware.CORS(s.cors))
		}

		r.Post("/auth/token", a.PostToken)
		r.Post("/auth/refresh", a.PostRefresh)

		r.Group(func(r chi.Router) {
			r.Use(auth.BearerTokenMiddleware(s.store))

			r.Get("/auth/me", a.GetMe)
			r.Delete("/auth/token", a.DeleteToken)

			r.Get("/projects", a.ListProjects)
			r.Post("/projects", a.CreateProject)
			r.Get("/projects/{key}", a.GetProject)
			r.Patch("/projects/{key}", a.UpdateProject)
			r.Delete("/projects/{key}", a.DeleteProject)
			r.Get("/projects/{key}/members", a.GetMembers)
			r.Post("/projects/{key}/members", a.AddMember)

			r.Get("/projects/{key}/issues", a.ListIssues)
			r.Post("/projects/{key}/issues", a.CreateIssue)
			r.Get("/projects/{key}/issues/{number}", a.GetIssue)
			r.Patch("/projects/{key}/issues/{number}", a.UpdateIssue)
			r.Delete("/projects/{key}/issues/{number}", a.DeleteIssue)
			r.Get("/projects/{key}/board", a.GetBoard)
			r.Patch("/projects/{key}/board/move", a.MoveIssue)

			r.Get("/issues/{id}/comments", a.ListComments)
			r.Post("/issues/{id}/comments", a.CreateComment)
			r.Patch("/comments/{id}", a.UpdateComment)
			r.Delete("/comments/{id}", a.DeleteComment)

			r.Get("/projects/{key}/labels", a.ListLabels)
			r.Post("/projects/{key}/labels", a.CreateLabel)
			r.Patch("/labels/{id}", a.UpdateLabel)
			r.Delete("/labels/{id}", a.DeleteLabel)

			r.Get("/projects/{key}/tags", a.ListProjectTags)
			r.Post("/projects/{key}/tags", a.AddProjectTag)
			r.Delete("/projects/{key}/tags/{tag}", a.RemoveProjectTag)
			r.Get("/tags", a.ListAllTags)
			r.Get("/tags/groups", a.ListTagGroups)

			r.Post("/time/start", a.StartTimer)
			r.Post("/time/stop", a.StopTimer)
			r.Get("/time/active", a.GetActiveTimer)
			r.Get("/issues/{id}/time", a.ListTimeEntries)
			r.Post("/issues/{id}/time", a.CreateTimeEntry)
			r.Patch("/time/{id}", a.UpdateTimeEntry)
			r.Delete("/time/{id}", a.DeleteTimeEntry)
			r.Get("/time/sheet", a.GetTimesheet)

			r.Get("/projects/{key}/analytics/velocity", a.GetVelocity)
			r.Get("/projects/{key}/analytics/cycle-time", a.GetCycleTime)
			r.Get("/projects/{key}/analytics/estimates", a.GetEstimates)
			r.Get("/projects/{key}/analytics/time-spent", a.GetTimeSpent)
			r.Get("/projects/{key}/analytics/health", a.GetHealth)
			r.Get("/analytics/compare", a.Compare)
			r.Get("/analytics/predict", a.Predict)
		})
	})

	// MCP over Streamable HTTP -- mounted outside chi's middleware stack
	// so that Timeout, BodyLimit, and CSRF middleware don't interfere
	// with long-lived streaming connections.
	if s.mcpServer != nil {
		mcpTransport := mcpgo.NewStreamableHTTPServer(s.mcpServer,
			mcpgo.WithHTTPContextFunc(s.mcpContextFunc()),
		)

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/mcp" {
				mcpTransport.ServeHTTP(w, req)
				return
			}
			r.ServeHTTP(w, req)
		})
	}

	return r
}

