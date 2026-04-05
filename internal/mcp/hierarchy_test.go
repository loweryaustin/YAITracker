package mcpserver

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestParentAssignCreatesCycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "CYC")

	mk := func(title string, parentID *string) *model.Issue {
		iss := &model.Issue{
			ProjectID:  p.ID,
			Title:      title,
			Type:       "task",
			Status:     "backlog",
			Priority:   "medium",
			ReporterID: u.ID,
			ParentID:   parentID,
		}
		if err := st.CreateIssue(ctx, iss); err != nil {
			t.Fatalf("CreateIssue %s: %v", title, err)
		}
		return iss
	}

	a := mk("a", nil)
	b := mk("b", &a.ID)
	c := mk("c", &b.ID)

	cycle, err := parentAssignCreatesCycle(ctx, st, a.ID, c.ID)
	if err != nil {
		t.Fatalf("parentAssignCreatesCycle: %v", err)
	}
	if !cycle {
		t.Fatal("expected cycle when a's parent would be c")
	}

	ok, err := parentAssignCreatesCycle(ctx, st, c.ID, a.ID)
	if err != nil {
		t.Fatalf("parentAssignCreatesCycle: %v", err)
	}
	if ok {
		t.Fatal("did not expect cycle for valid parent a on c")
	}
}

func TestParentIDFromOptionalNumber(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "PIN")
	parent := testutil.SeedIssue(t, st, p.ID, u.ID)

	pid, err := parentIDFromOptionalNumber(ctx, st, p.ID, 0)
	if err != nil || pid != nil {
		t.Fatalf("parentIDFromOptionalNumber 0: pid=%v err=%v", pid, err)
	}
	pid, err = parentIDFromOptionalNumber(ctx, st, p.ID, parent.Number)
	if err != nil || pid == nil || *pid != parent.ID {
		t.Fatalf("parentIDFromOptionalNumber: pid=%v err=%v", pid, err)
	}
	_, err = parentIDFromOptionalNumber(ctx, st, p.ID, 999)
	if err == nil {
		t.Fatal("expected error for missing parent number")
	}
}
