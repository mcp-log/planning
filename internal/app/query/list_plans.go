package query

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
	"github.com/mcp-log/planning/pkg/pagination"
)

// ListPlansQuery carries the filtering and pagination parameters for listing plans
type ListPlansQuery struct {
	Status   *plan.Status
	Mode     *plan.Mode
	Priority *plan.Priority
	Cursor   string
	Limit    int
}

// ListPlansResult holds the paginated result of a list plans query
type ListPlansResult struct {
	Plans      []*plan.Plan
	NextCursor string
	HasMore    bool
}

// ListPlansHandler handles the ListPlans query
type ListPlansHandler struct {
	repo plan.Repository
}

// NewListPlansHandler creates a new ListPlansHandler
func NewListPlansHandler(repo plan.Repository) *ListPlansHandler {
	return &ListPlansHandler{
		repo: repo,
	}
}

// Handle executes the ListPlans query with cursor-based pagination.
// The limit is clamped between the default (20) and maximum (100) values.
// An extra record is fetched to determine whether more results exist beyond the current page.
func (h *ListPlansHandler) Handle(ctx context.Context, q ListPlansQuery) (ListPlansResult, error) {
	page := pagination.NewPage(q.Cursor, q.Limit)

	filter := plan.ListFilter{
		Status:   q.Status,
		Mode:     q.Mode,
		Priority: q.Priority,
	}

	// Fetch limit+1 to detect next page
	plans, nextCursor, err := h.repo.List(ctx, filter, page.Limit+1, page.Cursor)
	if err != nil {
		return ListPlansResult{}, err
	}

	hasMore := len(plans) > page.Limit
	if hasMore {
		plans = plans[:page.Limit]
	}

	// Only set NextCursor when there are more results
	cursor := ""
	if hasMore {
		if nextCursor != "" {
			cursor = nextCursor
		} else if len(plans) > 0 {
			cursor = pagination.EncodeCursor(plans[len(plans)-1].ID)
		}
	}

	return ListPlansResult{
		Plans:      plans,
		NextCursor: cursor,
		HasMore:    hasMore,
	}, nil
}
