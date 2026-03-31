package plan

// Status represents the current state of a plan in its lifecycle
type Status string

const (
	// Created means plan is newly created and items can be added
	Created Status = "CREATED"

	// Processing means plan is being validated/processed (WAVE mode only)
	Processing Status = "PROCESSING"

	// Held means processing is paused (WAVE mode only)
	Held Status = "HELD"

	// Released means plan is released to warehouse floor (work available)
	Released Status = "RELEASED"

	// Completed means all work is finished (terminal state)
	Completed Status = "COMPLETED"

	// Cancelled means plan was cancelled with reason (terminal state)
	Cancelled Status = "CANCELLED"
)

// validTransitions defines the allowed state transitions
var validTransitions = map[Status][]Status{
	Created:    {Processing, Released, Cancelled}, // Released only for DYNAMIC auto-release
	Processing: {Held, Released, Cancelled},
	Held:       {Processing, Cancelled},
	Released:   {Completed, Cancelled},
	Completed:  {}, // Terminal state
	Cancelled:  {}, // Terminal state
}

// CanTransitionTo returns true if the transition from -> to is valid
func CanTransitionTo(from, to Status) bool {
	allowedTargets, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, target := range allowedTargets {
		if target == to {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the status is a terminal state
func (s Status) IsTerminal() bool {
	return s == Completed || s == Cancelled
}

// IsValid returns true if the status value is recognized
func (s Status) IsValid() bool {
	switch s {
	case Created, Processing, Held, Released, Completed, Cancelled:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}
