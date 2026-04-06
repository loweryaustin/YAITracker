package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/model"
)

const lockoutThreshold = 10
const lockoutDuration = 15 * time.Minute

func (h *Handler) GetLogin(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to dashboard
	if sessionID := auth.GetSessionID(r); sessionID != "" {
		if _, err := h.Store.GetSession(r.Context(), sessionID); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}

	userCount, err := h.Store.CountUsers(r.Context())
	if err != nil {
		log.Printf("count users: %v", err)
	}
	if userCount == 0 {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	h.renderLogin(w, r, "", "")
}

func (h *Handler) PostLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		h.renderLogin(w, r, "Invalid form data", "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.renderLogin(w, r, "Email and password are required", email)
		return
	}

	user, err := h.Store.GetUserByEmail(r.Context(), email)
	if err != nil {
		h.renderLogin(w, r, "Invalid email or password", email)
		return
	}

	if user.IsLocked() {
		remaining := time.Until(*user.LockedUntil).Minutes()
		h.renderLogin(w, r, "Account locked. Try again in "+formatMinutes(remaining), email)
		return
	}

	if !auth.CheckPassword(user.Password, password) {
		count, err := h.Store.IncrementFailedAttempts(r.Context(), user.ID)
		if err != nil {
			log.Printf("increment failed attempts: %v", err)
		}
		if count >= lockoutThreshold {
			if err := h.Store.LockUser(r.Context(), user.ID, time.Now().UTC().Add(lockoutDuration)); err != nil {
				log.Printf("lock user: %v", err)
			}
			h.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit log
				EntityType: "auth", UserID: user.ID, Action: "account_locked",
				IPAddress: h.clientIP(r),
			})
		}
		h.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit log
			EntityType: "auth", UserID: user.ID, Action: "login_failed",
			IPAddress: h.clientIP(r),
		})
		h.renderLogin(w, r, "Invalid email or password", email)
		return
	}

	if err := h.Store.ResetFailedAttempts(r.Context(), user.ID); err != nil {
		log.Printf("reset failed attempts: %v", err)
	}

	sess, err := h.Store.CreateSession(r.Context(), user.ID, auth.SessionDuration)
	if err != nil {
		h.renderLogin(w, r, "Internal error", email)
		return
	}

	h.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit log
		EntityType: "auth", UserID: user.ID, Action: "login_success",
		IPAddress: h.clientIP(r),
	})

	auth.SetSessionCookie(w, sess.ID, h.Secure)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) GetRegister(w http.ResponseWriter, r *http.Request) {
	userCount, err := h.Store.CountUsers(r.Context())
	if err != nil {
		log.Printf("count users: %v", err)
	}
	if userCount > 0 {
		// Only allow registration if no users exist (first-run) or via invite
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.renderRegister(w, r, "", true)
}

func (h *Handler) PostRegister(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	userCount, err := h.Store.CountUsers(r.Context())
	if err != nil {
		log.Printf("count users: %v", err)
	}
	if userCount > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderRegister(w, r, "Invalid form data", true)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if name == "" || email == "" || password == "" {
		h.renderRegister(w, r, "All fields are required", true)
		return
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		h.renderRegister(w, r, err.Error(), true)
		return
	}

	user := &model.User{
		Email:    email,
		Name:     name,
		Password: hashedPassword,
		Role:     "admin", // First user is admin
	}

	if err := h.Store.CreateUser(r.Context(), user); err != nil {
		h.renderRegister(w, r, "Could not create account: "+err.Error(), true)
		return
	}

	sess, err := h.Store.CreateSession(r.Context(), user.ID, auth.SessionDuration)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.Store.LogActivity(r.Context(), &model.ActivityLog{ //nolint:errcheck // best-effort audit log
		EntityType: "auth", UserID: user.ID, Action: "register",
		IPAddress: h.clientIP(r),
	})

	auth.SetSessionCookie(w, sess.ID, h.Secure)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) PostLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := auth.GetSessionID(r)
	if sessionID != "" {
		if err := h.Store.DeleteSession(r.Context(), sessionID); err != nil {
			log.Printf("delete session: %v", err)
		}
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func formatMinutes(m float64) string {
	if m < 1 {
		return "less than a minute"
	}
	return strconv.Itoa(int(m)) + " minutes"
}
