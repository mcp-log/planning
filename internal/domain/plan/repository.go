package plan

import (
	"context"
)

// Repository defines the persistence interface for the Plan aggregate
type Repository interface {
	// Save creates a new plan with its items in a single transaction
	Save(ctx context.Context, plan *Plan) error

	// FindByID retrieves a plan by ID with all its items
	// Returns ErrPlanNotFound if plan does not exist
	FindByID(ctx context.Context, id string) (*Plan, error)

	// Update persists changes to an existing plan
	// Items are managed separately via Save (on first creation)
	Update(ctx context.Context, plan *Plan) error

	// List retrieves plans matching filter criteria with cursor-based pagination
	// Returns plans, nextCursor (empty string if no more pages), and error
	List(ctx context.Context, filter ListFilter, limit int, cursor string) ([]*Plan, string, error)
}

// ListFilter defines optional filters for querying plans
type ListFilter struct {
	Status   *Status   // Optional: filter by status
	Mode     *Mode     // Optional: filter by mode
	Priority *Priority // Optional: filter by priority
}
