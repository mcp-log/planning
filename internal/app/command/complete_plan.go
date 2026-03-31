package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// CompletePlanHandler handles the CompletePlan command
type CompletePlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewCompletePlanHandler creates a new CompletePlanHandler
func NewCompletePlanHandler(repo plan.Repository, publisher EventPublisher) *CompletePlanHandler {
	return &CompletePlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the CompletePlan command
func (h *CompletePlanHandler) Handle(ctx context.Context, planID string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Complete the plan
	if err := p.Complete(); err != nil {
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
