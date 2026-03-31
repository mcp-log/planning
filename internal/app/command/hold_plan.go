package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// HoldPlanHandler handles the HoldPlan command
type HoldPlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewHoldPlanHandler creates a new HoldPlanHandler
func NewHoldPlanHandler(repo plan.Repository, publisher EventPublisher) *HoldPlanHandler {
	return &HoldPlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the HoldPlan command
func (h *HoldPlanHandler) Handle(ctx context.Context, planID string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Hold the plan
	if err := p.Hold(); err != nil {
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
