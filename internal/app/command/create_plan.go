package command

import (
	"context"

	"github.com/mcp-log/planning/internal/domain/plan"
	"github.com/mcp-log/planning/pkg/events"
)

// EventPublisher publishes domain events to the message broker
type EventPublisher interface {
	Publish(ctx context.Context, event events.DomainEvent) error
	PublishBatch(ctx context.Context, events []events.DomainEvent) error
}

// CreatePlanHandler handles the CreatePlan command
type CreatePlanHandler struct {
	repo      plan.Repository
	publisher EventPublisher
}

// NewCreatePlanHandler creates a new CreatePlanHandler
func NewCreatePlanHandler(repo plan.Repository, publisher EventPublisher) *CreatePlanHandler {
	return &CreatePlanHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the CreatePlan command
func (h *CreatePlanHandler) Handle(ctx context.Context, name string, mode plan.Mode, groupingStrategy plan.Strategy, priority plan.Priority, maxItems int, notes string) (*plan.Plan, error) {
	// Create the plan aggregate
	p, err := plan.NewPlan(name, mode, groupingStrategy, priority, maxItems, notes)
	if err != nil {
		return nil, err
	}

	// Persist the plan
	if err := h.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	// Publish domain events
	if err := h.publisher.PublishBatch(ctx, p.DomainEvents()); err != nil {
		// Log error but don't fail the command (eventual consistency)
		// In production, this would use a proper logger
	}

	// Clear events after publishing
	p.ClearEvents()

	return p, nil
}
