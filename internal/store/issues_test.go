package store_test

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestCreateIssue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		issue      model.Issue
		wantErr    bool
		wantNumber int
	}{
		{
			name: "valid task",
			issue: model.Issue{
				Title:    "Test issue",
				Type:     "task",
				Status:   "backlog",
				Priority: "medium",
			},
			wantNumber: 1,
		},
		{
			name: "auto-increments number",
			issue: model.Issue{
				Title:    "Second issue",
				Type:     "bug",
				Status:   "todo",
				Priority: "high",
			},
			wantNumber: 2,
		},
	}

	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "ISS")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &model.Issue{
				ProjectID:  p.ID,
				Title:      tt.issue.Title,
				Type:       tt.issue.Type,
				Status:     tt.issue.Status,
				Priority:   tt.issue.Priority,
				ReporterID: u.ID,
			}

			err := st.CreateIssue(ctx, issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if issue.ID == "" {
					t.Error("CreateIssue() did not set ID")
				}
				if issue.Number != tt.wantNumber {
					t.Errorf("CreateIssue() Number = %d, want %d", issue.Number, tt.wantNumber)
				}
			}
		})
	}
}

func TestGetIssueByNumber(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "GET")

	issue := &model.Issue{
		ProjectID:  p.ID,
		Title:      "Find me",
		Type:       "task",
		Status:     "backlog",
		Priority:   "none",
		ReporterID: u.ID,
	}
	if err := st.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("seed issue: %v", err)
	}

	got, err := st.GetIssueByNumber(ctx, p.ID, 1)
	if err != nil {
		t.Fatalf("GetIssueByNumber() error = %v", err)
	}
	if got.Title != "Find me" {
		t.Errorf("GetIssueByNumber() Title = %v, want 'Find me'", got.Title)
	}
}

func TestGetIssueByNumber_notFound(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, _ := testutil.SeedProject(t, st, "NF")

	_, err := st.GetIssueByNumber(ctx, p.ID, 999)
	if err == nil {
		t.Error("GetIssueByNumber() expected error for missing issue, got nil")
	}
}

func TestCreateIssue_setsStartedAt(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "START")

	issue := &model.Issue{
		ProjectID:  p.ID,
		Title:      "In progress from start",
		Type:       "task",
		Status:     "in_progress",
		Priority:   "medium",
		ReporterID: u.ID,
	}
	if err := st.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}
	if issue.StartedAt == nil {
		t.Error("CreateIssue() with in_progress status did not set StartedAt")
	}
}

func TestCreateIssue_setsCompletedAt(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "DONE")

	issue := &model.Issue{
		ProjectID:  p.ID,
		Title:      "Done from start",
		Type:       "task",
		Status:     "done",
		Priority:   "low",
		ReporterID: u.ID,
	}
	if err := st.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}
	if issue.CompletedAt == nil {
		t.Error("CreateIssue() with done status did not set CompletedAt")
	}
}

func TestListIssues(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()
	p, u := testutil.SeedProject(t, st, "LIST")

	for i := 0; i < 3; i++ {
		issue := &model.Issue{
			ProjectID:  p.ID,
			Title:      "Issue",
			Type:       "task",
			Status:     "backlog",
			Priority:   "none",
			ReporterID: u.ID,
		}
		if err := st.CreateIssue(ctx, issue); err != nil {
			t.Fatalf("seed issue %d: %v", i, err)
		}
	}

	issues, total, err := st.ListIssues(ctx, model.IssueFilter{
		ProjectID: p.ID,
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if total != 3 {
		t.Errorf("ListIssues() total = %d, want 3", total)
	}
	if len(issues) != 3 {
		t.Errorf("ListIssues() returned %d issues, want 3", len(issues))
	}
}

func TestMapIssueIDToNumberAndChildNumbers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, u := testutil.SeedProject(t, st, "MAP")

	parent := &model.Issue{
		ProjectID:  p.ID,
		Title:      "parent",
		Type:       "task",
		Status:     "backlog",
		Priority:   "medium",
		ReporterID: u.ID,
	}
	if err := st.CreateIssue(ctx, parent); err != nil {
		t.Fatalf("CreateIssue parent: %v", err)
	}
	child := &model.Issue{
		ProjectID:  p.ID,
		Title:      "child",
		Type:       "task",
		Status:     "backlog",
		Priority:   "medium",
		ReporterID: u.ID,
		ParentID:   &parent.ID,
	}
	if err := st.CreateIssue(ctx, child); err != nil {
		t.Fatalf("CreateIssue child: %v", err)
	}

	idToNum, err := st.MapIssueIDToNumber(ctx, p.ID)
	if err != nil {
		t.Fatalf("MapIssueIDToNumber: %v", err)
	}
	if idToNum[parent.ID] != parent.Number || idToNum[child.ID] != child.Number {
		t.Fatalf("MapIssueIDToNumber = %v, want parent %d child %d", idToNum, parent.Number, child.Number)
	}

	byParent, err := st.MapParentIDToChildNumbers(ctx, p.ID)
	if err != nil {
		t.Fatalf("MapParentIDToChildNumbers: %v", err)
	}
	kids := byParent[parent.ID]
	if len(kids) != 1 || kids[0] != child.Number {
		t.Fatalf("child numbers = %v, want [%d]", kids, child.Number)
	}
}
