package store_test

import (
	"context"
	"testing"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/testutil"
)

func TestGetLabelByName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.NewTestStore(t)
	p, _ := testutil.SeedProject(t, st, "LBL")

	l := &model.Label{ProjectID: p.ID, Name: "My-Label", Color: "#ff0000"}
	if err := st.CreateLabel(ctx, l); err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}

	got, err := st.GetLabelByName(ctx, p.ID, "my-label")
	if err != nil {
		t.Fatalf("GetLabelByName: %v", err)
	}
	if got == nil || got.ID != l.ID || got.Name != "My-Label" {
		t.Fatalf("GetLabelByName = %+v, want id %s name My-Label", got, l.ID)
	}

	miss, err := st.GetLabelByName(ctx, p.ID, "nope")
	if err != nil || miss != nil {
		t.Fatalf("GetLabelByName missing: %+v err=%v", miss, err)
	}
}
