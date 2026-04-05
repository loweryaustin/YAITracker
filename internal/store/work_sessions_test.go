package store_test

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestCreateWorkSession(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	u := testutil.SeedUser(t, st)

	ws, err := st.CreateWorkSession(ctx, u.ID, "morning dev block")
	if err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}
	if ws.ID == "" {
		t.Error("CreateWorkSession() did not set ID")
	}
	if ws.UserID != u.ID {
		t.Errorf("CreateWorkSession() UserID = %v, want %v", ws.UserID, u.ID)
	}
	if ws.Description != "morning dev block" {
		t.Errorf("CreateWorkSession() Description = %v, want 'morning dev block'", ws.Description)
	}
	if ws.StartedAt.IsZero() {
		t.Error("CreateWorkSession() did not set StartedAt")
	}
	if ws.EndedAt != nil {
		t.Error("CreateWorkSession() should not set EndedAt")
	}
}

func TestCreateWorkSession_duplicateRejects(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	u := testutil.SeedUser(t, st)

	if _, err := st.CreateWorkSession(ctx, u.ID, "first"); err != nil {
		t.Fatalf("first CreateWorkSession() error = %v", err)
	}

	_, err := st.CreateWorkSession(ctx, u.ID, "second")
	if err == nil {
		t.Error("CreateWorkSession() should reject duplicate active session")
	}
}

func TestGetActiveWorkSession(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	u := testutil.SeedUser(t, st)

	got, err := st.GetActiveWorkSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveWorkSession() error = %v", err)
	}
	if got != nil {
		t.Error("GetActiveWorkSession() should return nil when no session active")
	}

	ws, err := st.CreateWorkSession(ctx, u.ID, "test")
	if err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}

	got, err = st.GetActiveWorkSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveWorkSession() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetActiveWorkSession() returned nil, want session")
	}
	if got.ID != ws.ID {
		t.Errorf("GetActiveWorkSession() ID = %v, want %v", got.ID, ws.ID)
	}
}

func TestEndWorkSession(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	u := testutil.SeedUser(t, st)

	if _, err := st.CreateWorkSession(ctx, u.ID, "ending soon"); err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}

	ws, err := st.EndWorkSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("EndWorkSession() error = %v", err)
	}
	if ws.EndedAt == nil {
		t.Error("EndWorkSession() did not set EndedAt")
	}
	if ws.Duration == nil {
		t.Error("EndWorkSession() did not set Duration")
	}

	got, err := st.GetActiveWorkSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveWorkSession() error = %v", err)
	}
	if got != nil {
		t.Error("GetActiveWorkSession() should return nil after ending session")
	}
}

func TestEndWorkSession_noActive(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	u := testutil.SeedUser(t, st)

	_, err := st.EndWorkSession(ctx, u.ID)
	if err == nil {
		t.Error("EndWorkSession() should error when no active session")
	}
}

func TestListRecentWorkSessions(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	u := testutil.SeedUser(t, st)

	ws1, _ := st.CreateWorkSession(ctx, u.ID, "first")
	st.EndWorkSession(ctx, u.ID)

	ws2, _ := st.CreateWorkSession(ctx, u.ID, "second")
	st.EndWorkSession(ctx, u.ID)

	sessions, err := st.ListRecentWorkSessions(ctx, u.ID, 10)
	if err != nil {
		t.Fatalf("ListRecentWorkSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListRecentWorkSessions() returned %d, want 2", len(sessions))
	}
	if sessions[0].ID != ws2.ID {
		t.Error("ListRecentWorkSessions() should return most recent first")
	}
	if sessions[1].ID != ws1.ID {
		t.Error("ListRecentWorkSessions() second entry should be older session")
	}
}

func TestGetSessionUtilization(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "UTIL")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	ws, err := st.CreateWorkSession(ctx, u.ID, "utilization test")
	if err != nil {
		t.Fatalf("CreateWorkSession: %v", err)
	}
	if _, err := st.StartTimer(ctx, issue.ID, u.ID, "human", ws.ID, "", ""); err != nil {
		t.Fatalf("StartTimer: %v", err)
	}
	st.StopTimer(ctx, u.ID)
	st.EndWorkSession(ctx, u.ID)

	util, err := st.GetSessionUtilization(ctx, ws.ID)
	if err != nil {
		t.Fatalf("GetSessionUtilization() error = %v", err)
	}
	// Both session and timer were very short, so utilization should be >= 0
	if util < 0 {
		t.Errorf("GetSessionUtilization() = %v, want >= 0", util)
	}
}

func TestEndWorkSession_stopsHumanTimer(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "ENDT")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	ws, err := st.CreateWorkSession(ctx, u.ID, "session with timer")
	if err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}

	_, err = st.StartTimer(ctx, issue.ID, u.ID, "human", ws.ID, "", "")
	if err != nil {
		t.Fatalf("StartTimer() error = %v", err)
	}

	_, err = st.EndWorkSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("EndWorkSession() error = %v", err)
	}

	timers, err := st.GetActiveTimers(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveTimers() error = %v", err)
	}
	for _, te := range timers {
		if te.ActorType == "human" {
			t.Error("EndWorkSession() should have stopped the human timer")
		}
	}
}
