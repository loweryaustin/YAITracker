package mcpserver

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestStrictAgentWorkflow(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{"default strict", "", true},
		{"explicit true", "true", true},
		{"explicit 1", "1", true},
		{"explicit false", "false", false},
		{"explicit 0", "0", false},
		{"explicit no", "no", false},
		{"explicit off", "off", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("YAITRACKER_STRICT_AGENT_WORKFLOW", tc.env)
			} else {
				t.Setenv("YAITRACKER_STRICT_AGENT_WORKFLOW", "")
			}
			if got := StrictAgentWorkflow(); got != tc.want {
				t.Errorf("StrictAgentWorkflow() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestActiveAgentTimerOnIssue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "WFMC")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	ok, err := activeAgentTimerOnIssue(ctx, st, u.ID, issue.ID)
	if err != nil {
		t.Fatalf("activeAgentTimerOnIssue: %v", err)
	}
	if ok {
		t.Fatal("expected no active agent timer")
	}

	if _, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", ""); err != nil {
		t.Fatalf("StartTimer: %v", err)
	}

	ok, err = activeAgentTimerOnIssue(ctx, st, u.ID, issue.ID)
	if err != nil {
		t.Fatalf("activeAgentTimerOnIssue: %v", err)
	}
	if !ok {
		t.Fatal("expected active agent timer on issue")
	}
}

func TestBeginAgentWorkStartsTimer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "BGAW")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	iss, entry, _, err := beginAgentWork(ctx, st, u.ID, p.Key, issue.Number, "")
	if err != nil {
		t.Fatalf("beginAgentWork: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected timer id")
	}
	if iss.Title != issue.Title {
		t.Errorf("issue title = %q", iss.Title)
	}

	ok, err := activeAgentTimerOnIssue(ctx, st, u.ID, issue.ID)
	if err != nil {
		t.Fatalf("activeAgentTimerOnIssue: %v", err)
	}
	if !ok {
		t.Fatal("beginAgentWork should leave active agent timer")
	}
}

func TestBeginAgentWorkTwoIssuesTwoAgentTimers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "TWOI")
	a := testutil.SeedIssue(t, st, p.ID, u.ID)
	b := testutil.SeedIssue(t, st, p.ID, u.ID)

	if _, _, _, err := beginAgentWork(ctx, st, u.ID, p.Key, a.Number, ""); err != nil {
		t.Fatalf("beginAgentWork A: %v", err)
	}
	if _, _, _, err := beginAgentWork(ctx, st, u.ID, p.Key, b.Number, ""); err != nil {
		t.Fatalf("beginAgentWork B: %v", err)
	}

	timers, err := st.GetActiveTimers(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetActiveTimers: %v", err)
	}
	var agentCount int
	for i := range timers {
		if timers[i].ActorType == "agent" {
			agentCount++
		}
	}
	if agentCount != 2 {
		t.Fatalf("want 2 active agent timers, got %d", agentCount)
	}

	okA, err := activeAgentTimerOnIssue(ctx, st, u.ID, a.ID)
	if err != nil || !okA {
		t.Fatalf("issue A should have agent timer: ok=%v err=%v", okA, err)
	}
	okB, err := activeAgentTimerOnIssue(ctx, st, u.ID, b.ID)
	if err != nil || !okB {
		t.Fatalf("issue B should have agent timer: ok=%v err=%v", okB, err)
	}
}

func TestStoppingTimerOnOneIssueLeavesOtherIssueTimer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "STOP")
	a := testutil.SeedIssue(t, st, p.ID, u.ID)
	b := testutil.SeedIssue(t, st, p.ID, u.ID)

	_, entryA, _, err := beginAgentWork(ctx, st, u.ID, p.Key, a.Number, "")
	if err != nil {
		t.Fatalf("beginAgentWork A: %v", err)
	}
	if _, _, _, err := beginAgentWork(ctx, st, u.ID, p.Key, b.Number, ""); err != nil {
		t.Fatalf("beginAgentWork B: %v", err)
	}

	if _, err := st.StopTimerByID(ctx, entryA.ID); err != nil {
		t.Fatalf("StopTimerByID A: %v", err)
	}

	okA, err := activeAgentTimerOnIssue(ctx, st, u.ID, a.ID)
	if err != nil {
		t.Fatalf("activeAgentTimerOnIssue A: %v", err)
	}
	if okA {
		t.Fatal("issue A timer should be stopped")
	}
	okB, err := activeAgentTimerOnIssue(ctx, st, u.ID, b.ID)
	if err != nil || !okB {
		t.Fatalf("issue B should still have agent timer: ok=%v err=%v", okB, err)
	}
}
