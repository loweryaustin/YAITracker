// Package testutil provides shared test helpers for YAITracker tests.
package testutil

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	yaitracker "yaitracker.com/loweryaustin"
	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
)

// migrateMu serializes calls to store.Migrate because goose uses package-level
// globals (SetBaseFS, SetDialect) that race when called from parallel tests.
var migrateMu sync.Mutex

// NewTestStore creates an in-memory SQLite store with migrations applied.
// It registers a cleanup function to close the store when the test finishes.
func NewTestStore(t *testing.T) *store.Store {
	t.Helper()

	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("testutil.NewTestStore: open: %v", err)
	}

	migrateMu.Lock()
	err = store.Migrate(st.DB(), yaitracker.MigrationsFS, "migrations")
	migrateMu.Unlock()

	if err != nil {
		st.Close()
		t.Fatalf("testutil.NewTestStore: migrate: %v", err)
	}

	t.Cleanup(func() { st.Close() })
	return st
}

// NewTestStoreFile creates a file-backed SQLite store in t.TempDir() with migrations
// applied. Use instead of NewTestStore when a test needs concurrent connections to
// the same DB (:memory: uses a separate empty DB per connection with the default pool).
func NewTestStoreFile(t *testing.T) *store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "yaitracker-test.db")
	st, err := store.New(path)
	if err != nil {
		t.Fatalf("testutil.NewTestStoreFile: open: %v", err)
	}

	migrateMu.Lock()
	err = store.Migrate(st.DB(), yaitracker.MigrationsFS, "migrations")
	migrateMu.Unlock()

	if err != nil {
		//nolint:errcheck // best-effort close after failed migrate
		st.Close()
		t.Fatalf("testutil.NewTestStoreFile: migrate: %v", err)
	}

	t.Cleanup(func() {
		//nolint:errcheck // test teardown
		st.Close()
	})
	return st
}

// SeedUser creates a test user and returns it. The password field is set to a
// bcrypt hash of "testpassword123" but callers should not rely on the value.
func SeedUser(t *testing.T, st *store.Store) *model.User {
	t.Helper()

	u := &model.User{
		Email:    "test-" + store.NewID()[:8] + "@example.com",
		Name:     "Test User",
		Password: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012", // placeholder hash
		Role:     "admin",
	}

	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("testutil.SeedUser: %v", err)
	}
	return u
}

// SeedProject creates a test project with the given key and returns it.
// It also seeds a user as the project creator.
func SeedProject(t *testing.T, st *store.Store, key string) (*model.Project, *model.User) {
	t.Helper()

	u := SeedUser(t, st)
	p := &model.Project{
		Key:       key,
		Name:      "Test Project " + key,
		Status:    "active",
		CreatedBy: u.ID,
	}

	if err := st.CreateProject(context.Background(), p); err != nil {
		t.Fatalf("testutil.SeedProject: %v", err)
	}
	return p, u
}

// SeedIssue creates a test issue in the given project and returns it.
func SeedIssue(t *testing.T, st *store.Store, projectID, reporterID string) *model.Issue {
	t.Helper()

	issue := &model.Issue{
		ProjectID:  projectID,
		Title:      "Test Issue " + store.NewID()[:6],
		Type:       "task",
		Status:     "backlog",
		Priority:   "medium",
		ReporterID: reporterID,
	}

	if err := st.CreateIssue(context.Background(), issue); err != nil {
		t.Fatalf("testutil.SeedIssue: %v", err)
	}
	return issue
}
