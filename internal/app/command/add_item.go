package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
)

// AddItemHandler handles the AddItem command
type AddItemHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewAddItemHandler creates a new AddItemHandler
func NewAddItemHandler(repo plan.Repository, publisher EventPublisher) *AddItemHandler {
	return &AddItemHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the AddItem command and returns the newly added item
func (h *AddItemHandler) Handle(ctx context.Context, planID string, orderID string, sku string, quantity int) (*plan.PlanItem, error) {
	// Load the plan
	p, err := h.repo.FindByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Add item to plan
	if err := p.AddItem(orderID, sku, quantity); err != nil {
		return nil, err
	}

	// Get the newly added item (last item in the list)
	newItem := p.Items[len(p.Items)-1]

	// Persist changes
	if err := h.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	// Publish domain events
	if err := h.publisher.PublishBatch(ctx, p.DomainEvents()); err != nil {
		// Log error but don't fail the command
	}

	// Clear events
	p.ClearEvents()

	return &newItem, nil
}
