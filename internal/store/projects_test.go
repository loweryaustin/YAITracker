package store_test

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/model"
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
