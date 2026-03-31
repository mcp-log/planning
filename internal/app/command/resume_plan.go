package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// ResumePlanHandler handles the ResumePlan command
type ResumePlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewResumePlanHandler creates a new ResumePlanHandler
func NewResumePlanHandler(repo plan.Repository, publisher EventPublisher) *ResumePlanHandler {
	return &ResumePlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the ResumePlan command
func (h *ResumePlanHandler) Handle(ctx context.Context, planID string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Resume the plan
	if err := p.Resume(); err != nil {
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
