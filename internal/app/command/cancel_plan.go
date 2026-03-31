package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// CancelPlanHandler handles the CancelPlan command
type CancelPlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewCancelPlanHandler creates a new CancelPlanHandler
func NewCancelPlanHandler(repo plan.Repository, publisher EventPublisher) *CancelPlanHandler {
	return &CancelPlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the CancelPlan command
func (h *CancelPlanHandler) Handle(ctx context.Context, planID string, reason string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Cancel the plan
	if err := p.Cancel(reason); err != nil {
		return err
	}

	// Persist changes
	if err := h.repo.Update(ctx, p); err != nil {
		return err
	}

	// Publish domain events
	if err := h.publisher.PublishBatch(ctx, p.DomainEvents()); err != nil {
		// Log error but don't fail the command
	}

	// Clear events
	p.ClearEvents()

	return nil
}
