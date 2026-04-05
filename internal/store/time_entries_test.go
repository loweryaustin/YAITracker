package store_test

import (
	"context"
	"testing"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestStartTimer_agent(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "AGNT")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	te, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer(agent) error = %v", err)
	}
	if te.ID == "" {
		t.Error("StartTimer() did not set ID")
	}
	if te.ActorType != "agent" {
		t.Errorf("StartTimer() ActorType = %v, want 'agent'", te.ActorType)
	}
	if te.SessionID != "" {
		t.Error("StartTimer(agent) should not have a session_id")
	}
}

func TestStartTimer_human(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "HUMN")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	ws, err := st.CreateWorkSession(ctx, u.ID, "test session")
	if err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}

	te, err := st.StartTimer(ctx, issue.ID, u.ID, "human", ws.ID, "", "")
	if err != nil {
		t.Fatalf("StartTimer(human) error = %v", err)
	}
	if te.ActorType != "human" {
		t.Errorf("StartTimer() ActorType = %v, want 'human'", te.ActorType)
	}
	if te.SessionID != ws.ID {
		t.Errorf("StartTimer() SessionID = %v, want %v", te.SessionID, ws.ID)
	}
}

func TestStartTimer_humanRequiresSession(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "NSES")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	_, err := st.StartTimer(ctx, issue.ID, u.ID, "human", "", "", "")
	if err == nil {
		t.Error("StartTimer(human) should reject when no session_id provided")
	}
}

func TestStartTimer_humanAutoStopsPrevious(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "ASTP")
	issue1 := testutil.SeedIssue(t, st, p.ID, u.ID)
	issue2 := testutil.SeedIssue(t, st, p.ID, u.ID)

	ws, err := st.CreateWorkSession(ctx, u.ID, "auto-stop test")
	if err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}

	first, err := st.StartTimer(ctx, issue1.ID, u.ID, "human", ws.ID, "", "")
	if err != nil {
		t.Fatalf("StartTimer(first) error = %v", err)
	}

	_, err = st.StartTimer(ctx, issue2.ID, u.ID, "human", ws.ID, "", "")
	if err != nil {
		t.Fatalf("StartTimer(second) error = %v", err)
	}

	stopped, err := st.GetTimeEntry(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetTimeEntry() error = %v", err)
	}
	if stopped.EndedAt == nil {
		t.Error("StartTimer(human) should have auto-stopped the previous human timer")
	}
	if stopped.Duration == nil {
		t.Error("auto-stopped timer should have duration set")
	}
}

func TestStartTimer_concurrentAgentTimers(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "CONC")
	issue1 := testutil.SeedIssue(t, st, p.ID, u.ID)
	issue2 := testutil.SeedIssue(t, st, p.ID, u.ID)

	_, err := st.StartTimer(ctx, issue1.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer(agent, issue1) error = %v", err)
	}

	_, err = st.StartTimer(ctx, issue2.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer(agent, issue2) error = %v, want nil (concurrent agent timers allowed)", err)
	}

	timers, err := st.GetActiveTimers(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveTimers() error = %v", err)
	}
	if len(timers) != 2 {
		t.Errorf("GetActiveTimers() returned %d, want 2", len(timers))
	}
}

func TestStartTimer_rejectsDuplicateIssueAgent(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "DUPE")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	_, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer(first) error = %v", err)
	}

	_, err = st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", "")
	if err == nil {
		t.Error("StartTimer() should reject duplicate agent timer on same issue")
	}
}

func TestStopTimerByID(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "STOP")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	te, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer() error = %v", err)
	}

	stopped, err := st.StopTimerByID(ctx, te.ID)
	if err != nil {
		t.Fatalf("StopTimerByID() error = %v", err)
	}
	if stopped.EndedAt == nil {
		t.Error("StopTimerByID() did not set EndedAt")
	}
	if stopped.Duration == nil {
		t.Error("StopTimerByID() did not set Duration")
	}
	if *stopped.Duration < 0 {
		t.Errorf("StopTimerByID() Duration = %d, want >= 0", *stopped.Duration)
	}
}

func TestGetActiveTimers(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "ACTV")
	issue1 := testutil.SeedIssue(t, st, p.ID, u.ID)
	issue2 := testutil.SeedIssue(t, st, p.ID, u.ID)

	ws, err := st.CreateWorkSession(ctx, u.ID, "test")
	if err != nil {
		t.Fatalf("CreateWorkSession() error = %v", err)
	}
	if _, err := st.StartTimer(ctx, issue1.ID, u.ID, "human", ws.ID, "", ""); err != nil {
		t.Fatalf("StartTimer(human) error = %v", err)
	}
	if _, err := st.StartTimer(ctx, issue2.ID, u.ID, "agent", "", "", ""); err != nil {
		t.Fatalf("StartTimer(agent) error = %v", err)
	}

	timers, err := st.GetActiveTimers(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveTimers() error = %v", err)
	}

	// human auto-stopped when agent started? No -- human and agent are independent.
	// Actually, human auto-stop only applies to another HUMAN timer. Agent timers are independent.
	// So we should have 2 active timers.
	if len(timers) != 2 {
		t.Errorf("GetActiveTimers() returned %d, want 2 (human + agent)", len(timers))
	}
}

func TestStopOrphanedTimers(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "ORPH")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	te, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer() error = %v", err)
	}

	// Manually backdate the started_at to simulate an orphan
	_, err = st.DB().ExecContext(ctx,
		`UPDATE time_entries SET started_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-10*time.Hour), te.ID,
	)
	if err != nil {
		t.Fatalf("backdate timer: %v", err)
	}

	n, err := st.StopOrphanedTimers(ctx, 8*time.Hour)
	if err != nil {
		t.Fatalf("StopOrphanedTimers() error = %v", err)
	}
	if n != 1 {
		t.Errorf("StopOrphanedTimers() stopped %d, want 1", n)
	}

	stopped, err := st.GetTimeEntry(ctx, te.ID)
	if err != nil {
		t.Fatalf("GetTimeEntry() error = %v", err)
	}
	if stopped.EndedAt == nil {
		t.Error("orphaned timer should be stopped")
	}
}

func TestGetDailySummary(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "DSUM")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	// Create a completed manual entry so duration is guaranteed non-zero.
	now := time.Now().UTC()
	startedAt := now.Add(-30 * time.Minute)
	dur := int64(1800)
	if err := st.CreateManualTimeEntry(ctx, &model.TimeEntry{
		IssueID:   issue.ID,
		UserID:    u.ID,
		ActorType: "human",
		StartedAt: startedAt,
		EndedAt:   &now,
		Duration:  &dur,
	}); err != nil {
		t.Fatalf("CreateManualTimeEntry() error = %v", err)
	}

	summary, err := st.GetDailySummary(ctx, u.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("GetDailySummary() error = %v", err)
	}
	if summary.IssueCount != 1 {
		t.Errorf("GetDailySummary() IssueCount = %d, want 1", summary.IssueCount)
	}
	if summary.TotalSecs != 1800 {
		t.Errorf("GetDailySummary() TotalSecs = %d, want 1800", summary.TotalSecs)
	}
	if summary.HumanSecs != 1800 {
		t.Errorf("GetDailySummary() HumanSecs = %d, want 1800", summary.HumanSecs)
	}
}

func TestGetActiveTimersWithIssues(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "ATWI")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	if _, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", ""); err != nil {
		t.Fatalf("StartTimer: %v", err)
	}

	timers, err := st.GetActiveTimersWithIssues(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveTimersWithIssues() error = %v", err)
	}
	if len(timers) != 1 {
		t.Fatalf("GetActiveTimersWithIssues() returned %d, want 1", len(timers))
	}
	if timers[0].Issue == nil {
		t.Fatal("GetActiveTimersWithIssues() Issue should not be nil")
	}
	if timers[0].Issue.ProjectKey != "ATWI" {
		t.Errorf("GetActiveTimersWithIssues() ProjectKey = %v, want ATWI", timers[0].Issue.ProjectKey)
	}
	if timers[0].Issue.Title == "" {
		t.Error("GetActiveTimersWithIssues() Issue.Title should not be empty")
	}
}
