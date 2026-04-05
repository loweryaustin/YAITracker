package mcpserver

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestResolveOrCreateLabelConcurrent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.NewTestStoreFile(t)
	p, _ := testutil.SeedProject(t, st, "LCC")

	const n = 12
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lbl, err := resolveOrCreateLabel(ctx, st, p.ID, "shared-label", "#abc")
			if err != nil {
				errs <- err
				return
			}
			if lbl == nil || lbl.Name != "shared-label" {
				errs <- fmt.Errorf("unexpected label %+v", lbl)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	labels, err := st.ListLabels(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if len(labels) != 1 || labels[0].Name != "shared-label" {
		t.Fatalf("ListLabels = %+v, want one shared-label", labels)
	}
}
