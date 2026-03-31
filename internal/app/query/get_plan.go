package query

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// GetPlanHandler handles the GetPlan query
type GetPlanHandler struct {
	repo plan.Repository
}

// NewGetPlanHandler creates a new GetPlanHandler
func NewGetPlanHandler(repo plan.Repository) *GetPlanHandler {
	return &GetPlanHandler{
		repo: repo,
	}
}

// Handle executes the GetPlan query
func (h *GetPlanHandler) Handle(ctx context.Context, planID string) (*plan.Plan, error) {
	return h.repo.FindByID(ctx, planID)
}
