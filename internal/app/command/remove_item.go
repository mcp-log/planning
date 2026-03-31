package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// RemoveItemHandler handles the RemoveItem command
type RemoveItemHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewRemoveItemHandler creates a new RemoveItemHandler
func NewRemoveItemHandler(repo plan.Repository, publisher EventPublisher) *RemoveItemHandler {
	return &RemoveItemHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the RemoveItem command
func (h *RemoveItemHandler) Handle(ctx context.Context, planID string, itemID string) error {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return err
	}

	// Remove item from plan
	if err := p.RemoveItem(itemID); err != nil {
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
