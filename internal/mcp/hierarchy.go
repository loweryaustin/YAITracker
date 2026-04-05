package mcpserver

import (
	"context"
	"fmt"

	"yaitracker.com/loweryaustin/internal/store"
)

const maxParentWalk = 10000

// parentAssignCreatesCycle reports whether assigning parentID as the parent of issueID would create a cycle.
func parentAssignCreatesCycle(ctx context.Context, st *store.Store, issueID, parentID string) (bool, error) {
	cur := parentID
	for range maxParentWalk {
		if cur == issueID {
			return true, nil
		}
		p, err := st.GetIssue(ctx, cur)
		if err != nil {
			return false, err
		}
		if p.ParentID == nil {
			return false, nil
		}
		cur = *p.ParentID
	}
	return false, fmt.Errorf("issue parent chain exceeded %d ancestors", maxParentWalk)
}

// parentIDFromOptionalNumber resolves an optional parent issue number within the project to a parent id.
func parentIDFromOptionalNumber(ctx context.Context, st *store.Store, projectID string, parentNumber int) (*string, error) {
	if parentNumber <= 0 {
		return nil, nil
	}
	parent, err := st.GetIssueByNumber(ctx, projectID, parentNumber)
	if err != nil {
		return nil, fmt.Errorf("parent issue #%d not found in project", parentNumber)
	}
	pid := parent.ID
	return &pid, nil
}
