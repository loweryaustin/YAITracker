package store_test

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestCreateProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		project model.Project
		wantErr bool
	}{
		{
			name:    "valid project",
			project: model.Project{Key: "VALID", Name: "Valid Project", Status: "active"},
			wantErr: false,
		},
		{
			name:    "duplicate key",
			project: model.Project{Key: "DUPE", Name: "Duplicate", Status: "active"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st := testutil.NewTestStore(t)
			ctx := context.Background()
			u := testutil.SeedUser(t, st)

			if tt.name == "duplicate key" {
				first := &model.Project{Key: tt.project.Key, Name: "First", Status: "active", CreatedBy: u.ID}
				if err := st.CreateProject(ctx, first); err != nil {
					t.Fatalf("seed duplicate: %v", err)
				}
			}

			p := &model.Project{
				Key:       tt.project.Key,
				Name:      tt.project.Name,
				Status:    tt.project.Status,
				CreatedBy: u.ID,
			}
			err := st.CreateProject(ctx, p)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProject() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && p.ID == "" {
				t.Error("CreateProject() did not set ID")
			}
		})
	}
}

func TestGetProjectByKey(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	p, _ := testutil.SeedProject(t, st, "FIND")
	got, err := st.GetProjectByKey(ctx, "FIND")
	if err != nil {
		t.Fatalf("GetProjectByKey() error = %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("GetProjectByKey() ID = %v, want %v", got.ID, p.ID)
	}
	if got.Key != "FIND" {
		t.Errorf("GetProjectByKey() Key = %v, want FIND", got.Key)
	}
}

func TestGetProjectByKey_notFound(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	_, err := st.GetProjectByKey(ctx, "NOPE")
	if err == nil {
		t.Error("GetProjectByKey() expected error for missing key, got nil")
	}
}

func TestListProjects(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	testutil.SeedProject(t, st, "AAA")
	testutil.SeedProject(t, st, "BBB")

	projects, err := st.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("ListProjects() returned %d projects, want 2", len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	p, _ := testutil.SeedProject(t, st, "UPD")
	p.Name = "Updated Name"
	p.Description = "Updated description"

	if err := st.UpdateProject(ctx, p); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	got, err := st.GetProjectByKey(ctx, "UPD")
	if err != nil {
		t.Fatalf("GetProjectByKey() after update error = %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("UpdateProject() Name = %v, want 'Updated Name'", got.Name)
	}
}

func TestDeleteProject(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	p, _ := testutil.SeedProject(t, st, "DEL")
	if err := st.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}

	_, err := st.GetProjectByKey(ctx, "DEL")
	if err == nil {
		t.Error("GetProjectByKey() after delete expected error, got nil")
	}
}

func TestDeleteProject_cascadesChildData(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	p, u := testutil.SeedProject(t, st, "CASC")
	issue := testutil.SeedIssue(t, st, p.ID, u.ID)

	if err := st.CreateComment(ctx, &model.Comment{
		IssueID: issue.ID, AuthorID: u.ID, Body: "test comment",
	}); err != nil {
		t.Fatalf("CreateComment: %v", err)
	}

	te, err := st.StartTimer(ctx, issue.ID, u.ID, "agent", "", "", "")
	if err != nil {
		t.Fatalf("StartTimer: %v", err)
	}
	if _, err := st.StopTimerByID(ctx, te.ID); err != nil {
		t.Fatalf("StopTimerByID: %v", err)
	}

	if err := st.CreateLabel(ctx, &model.Label{
		ProjectID: p.ID, Name: "bug", Color: "#ff0000",
	}); err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}

	if err := st.AddProjectTag(ctx, p.ID, "backend", ""); err != nil {
		t.Fatalf("AddProjectTag: %v", err)
	}

	if err := st.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}

	assertRowCount(t, st, "projects", "id = ?", p.ID, 0)
	assertRowCount(t, st, "issues", "project_id = ?", p.ID, 0)
	assertRowCount(t, st, "comments", "issue_id = ?", issue.ID, 0)
	assertRowCount(t, st, "time_entries", "issue_id = ?", issue.ID, 0)
	assertRowCount(t, st, "labels", "project_id = ?", p.ID, 0)
	assertRowCount(t, st, "project_tags", "project_id = ?", p.ID, 0)
	assertRowCount(t, st, "project_members", "project_id = ?", p.ID, 0)
}

func TestDeleteProject_nonExistent(t *testing.T) {
	t.Parallel()
	st := testutil.NewTestStore(t)
	ctx := context.Background()

	err := st.DeleteProject(ctx, "nonexistent-id")
	if err != nil {
		t.Errorf("DeleteProject(nonexistent) should not error, got: %v", err)
	}
}

func assertRowCount(t *testing.T, st *store.Store, table, where, arg string, want int) {
	t.Helper()
	var got int
	err := st.DB().QueryRow("SELECT COUNT(*) FROM "+table+" WHERE "+where, arg).Scan(&got)
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Errorf("%s rows (where %s=%q): got %d, want %d", table, where, arg, got, want)
	}
}
