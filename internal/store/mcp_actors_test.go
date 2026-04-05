package store_test

import (
	"context"
	"testing"
	"time"

	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestCreateMCPActorAndGet(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	_, u := testutil.SeedProject(t, st, "MCPA")

	a, err := st.CreateMCPActor(ctx, u.ID, "desktop")
	if err != nil {
		t.Fatalf("CreateMCPActor: %v", err)
	}
	if a.ID == "" {
		t.Fatal("expected id")
	}
	if a.LastHeartbeatAt == nil {
		t.Fatal("expected last_heartbeat_at to be set on creation")
	}

	got, err := st.GetMCPActorForUser(ctx, u.ID, a.ID)
	if err != nil {
		t.Fatalf("GetMCPActorForUser: %v", err)
	}
	if got.Label != "desktop" {
		t.Errorf("label = %q", got.Label)
	}
	if got.LastHeartbeatAt == nil {
		t.Fatal("expected last_heartbeat_at on fetched actor")
	}
}

func TestRevokeMCPActor(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	_, u := testutil.SeedProject(t, st, "MCPR")
	a := testutil.SeedMCPActor(t, st, u.ID, "x")

	if err := st.RevokeMCPActor(ctx, u.ID, a.ID); err != nil {
		t.Fatalf("RevokeMCPActor: %v", err)
	}
	if _, err := st.GetMCPActorForUser(ctx, u.ID, a.ID); err == nil {
		t.Fatal("expected revoked actor to be invisible")
	}
}

func TestTouchActorHeartbeat(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	_, u := testutil.SeedProject(t, st, "HBTA")
	a := testutil.SeedMCPActor(t, st, u.ID, "hb-test")

	before, err := st.GetMCPActorForUser(ctx, u.ID, a.ID)
	if err != nil {
		t.Fatalf("get before: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := st.TouchActorHeartbeat(ctx, u.ID, a.ID); err != nil {
		t.Fatalf("TouchActorHeartbeat: %v", err)
	}

	after, err := st.GetMCPActorForUser(ctx, u.ID, a.ID)
	if err != nil {
		t.Fatalf("get after: %v", err)
	}
	if !after.LastHeartbeatAt.After(*before.LastHeartbeatAt) {
		t.Errorf("heartbeat not updated: before=%v after=%v", before.LastHeartbeatAt, after.LastHeartbeatAt)
	}
}

func TestTouchActorHeartbeat_revoked(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	_, u := testutil.SeedProject(t, st, "HBTR")
	a := testutil.SeedMCPActor(t, st, u.ID, "revoked-hb")

	if err := st.RevokeMCPActor(ctx, u.ID, a.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := st.TouchActorHeartbeat(ctx, u.ID, a.ID); err == nil {
		t.Fatal("expected error touching revoked actor heartbeat")
	}
}

func TestRevokeExpiredActors(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	_, u := testutil.SeedProject(t, st, "REXP")
	fresh := testutil.SeedMCPActor(t, st, u.ID, "fresh")
	stale := testutil.SeedMCPActor(t, st, u.ID, "stale")

	// Make the stale actor's heartbeat old by touching it, then backdating via raw SQL.
	db := st.DB()
	old := time.Now().UTC().Add(-20 * time.Minute)
	if _, err := db.ExecContext(ctx,
		`UPDATE mcp_actors SET last_heartbeat_at = ? WHERE id = ?`, old, stale.ID,
	); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	count, err := st.RevokeExpiredActors(ctx, 15*time.Minute)
	if err != nil {
		t.Fatalf("RevokeExpiredActors: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 revoked, got %d", count)
	}

	// Stale should be gone.
	if _, err := st.GetMCPActorForUser(ctx, u.ID, stale.ID); err == nil {
		t.Fatal("expected stale actor to be revoked")
	}

	// Fresh should still be alive.
	if _, err := st.GetMCPActorForUser(ctx, u.ID, fresh.ID); err != nil {
		t.Fatalf("fresh actor should still exist: %v", err)
	}
}

func TestRevokeExpiredActors_stopsTimers(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "RTMR")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)
	actor := testutil.SeedMCPActor(t, st, u.ID, "timer-test")

	te, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "test", actor.ID)
	if err != nil {
		t.Fatalf("StartTimer: %v", err)
	}

	// Backdate the actor heartbeat.
	db := st.DB()
	old := time.Now().UTC().Add(-20 * time.Minute)
	if _, err := db.ExecContext(ctx,
		`UPDATE mcp_actors SET last_heartbeat_at = ? WHERE id = ?`, old, actor.ID,
	); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	count, err := st.RevokeExpiredActors(ctx, 15*time.Minute)
	if err != nil {
		t.Fatalf("RevokeExpiredActors: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 revoked, got %d", count)
	}

	// The timer should now be stopped.
	entries, err := st.ListTimeEntries(ctx, issue.ID)
	if err != nil {
		t.Fatalf("ListTimeEntries: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.ID == te.ID {
			found = true
			if e.EndedAt == nil {
				t.Fatal("expected timer to be stopped after actor expiration")
			}
		}
	}
	if !found {
		t.Fatal("timer entry not found")
	}
}
