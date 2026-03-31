package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// ProcessPlanHandler handles the ProcessPlan command
type ProcessPlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewProcessPlanHandler creates a new ProcessPlanHandler
func NewProcessPlanHandler(repo plan.Repository, publisher EventPublisher) *ProcessPlanHandler {
	return &ProcessPlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the ProcessPlan command
func (h *ProcessPlanHandler) Handle(ctx context.Context, planID string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Process the plan
	if err := p.Process(); err != nil {
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
