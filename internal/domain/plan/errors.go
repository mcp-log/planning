package plan

import (
	"errors"
	"fmt"
)

// Domain errors
var (
	// ErrPlanNotFound is returned when a plan cannot be found
	ErrPlanNotFound = errors.New("plan not found")

	// ErrDuplicateItem is returned when adding an item that already exists in the plan
	ErrDuplicateItem = errors.New("duplicate item in plan")

	// ErrPlanFull is returned when adding an item would exceed the plan's capacity
	ErrPlanFull = errors.New("plan has reached maximum capacity")

	// ErrEmptyPlan is returned when attempting to process a plan with zero items
	ErrEmptyPlan = errors.New("cannot process plan with zero items")

	// ErrItemNotFound is returned when a plan item cannot be found
	ErrItemNotFound = errors.New("plan item not found")

	// ErrInvalidName is returned when a plan name is empty or whitespace-only
	ErrInvalidName = errors.New("plan name cannot be empty")

	// ErrCancelReasonRequired is returned when cancelling without a reason
	ErrCancelReasonRequired = errors.New("cancellation reason is required")

	// ErrInvalidQuantity is returned when item quantity is less than or equal to zero
	ErrInvalidQuantity = errors.New("item quantity must be greater than zero")

	// ErrItemsNotAllowed is returned when attempting to add/remove items in an invalid status
	ErrItemsNotAllowed = errors.New("cannot modify items in current plan status")
)

// ErrInvalidTransition represents an invalid state transition error
type ErrInvalidTransition struct {
	From    Status
	To      Status
	Message string
}

func (e ErrInvalidTransition) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("invalid transition from %s to %s: %s", e.From, e.To, e.Message)
	}
	return fmt.Sprintf("invalid transition from %s to %s", e.From, e.To)
}

// NewErrInvalidTransition creates a new invalid transition error
func NewErrInvalidTransition(from, to Status, message string) ErrInvalidTransition {
	return ErrInvalidTransition{
		From:    from,
		To:      to,
		Message: message,
	}
}
