package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// ReleasePlanHandler handles the ReleasePlan command
type ReleasePlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewReleasePlanHandler creates a new ReleasePlanHandler
func NewReleasePlanHandler(repo plan.Repository, publisher EventPublisher) *ReleasePlanHandler {
	return &ReleasePlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the ReleasePlan command
func (h *ReleasePlanHandler) Handle(ctx context.Context, planID string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Release the plan
	if err := p.Release(); err != nil {
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
