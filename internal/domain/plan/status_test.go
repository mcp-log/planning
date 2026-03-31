package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransitionTo_ValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		// From CREATED
		{"CREATED to PROCESSING", Created, Processing, true},
		{"CREATED to RELEASED", Created, Released, true}, // DYNAMIC auto-release
		{"CREATED to CANCELLED", Created, Cancelled, true},

		// From PROCESSING
		{"PROCESSING to HELD", Processing, Held, true},
		{"PROCESSING to RELEASED", Processing, Released, true},
		{"PROCESSING to CANCELLED", Processing, Cancelled, true},

		// From HELD
		{"HELD to PROCESSING", Held, Processing, true},
		{"HELD to CANCELLED", Held, Cancelled, true},

		// From RELEASED
		{"RELEASED to COMPLETED", Released, Completed, true},
		{"RELEASED to CANCELLED", Released, Cancelled, true},

		// Terminal states
		{"COMPLETED to any", Completed, Created, false},
		{"CANCELLED to any", Cancelled, Created, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransitionTo(tt.from, tt.to)
			assert.Equal(t, tt.want, got, "CanTransitionTo(%s, %s)", tt.from, tt.to)
		})
	}
}

func TestCanTransitionTo_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		// Invalid from CREATED
		{"CREATED to COMPLETED", Created, Completed},
		{"CREATED to HELD", Created, Held},

		// Invalid from PROCESSING
		{"PROCESSING to CREATED", Processing, Created},
		{"PROCESSING to COMPLETED", Processing, Completed},

		// Invalid from HELD
		{"HELD to CREATED", Held, Created},
		{"HELD to RELEASED", Held, Released},
		{"HELD to COMPLETED", Held, Completed},

		// Invalid from RELEASED
		{"RELEASED to CREATED", Released, Created},
		{"RELEASED to PROCESSING", Released, Processing},
		{"RELEASED to HELD", Released, Held},

		// Terminal states cannot transition
		{"COMPLETED to PROCESSING", Completed, Processing},
		{"COMPLETED to CANCELLED", Completed, Cancelled},
		{"CANCELLED to PROCESSING", Cancelled, Processing},
		{"CANCELLED to COMPLETED", Cancelled, Completed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransitionTo(tt.from, tt.to)
			assert.False(t, got, "Should not allow transition from %s to %s", tt.from, tt.to)
		})
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{Created, false},
		{Processing, false},
		{Held, false},
		{Released, false},
		{Completed, true},
		{Cancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsTerminal()
			assert.Equal(t, tt.want, got, "%s.IsTerminal()", tt.status)
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"CREATED is valid", Created, true},
		{"PROCESSING is valid", Processing, true},
		{"HELD is valid", Held, true},
		{"RELEASED is valid", Released, true},
		{"COMPLETED is valid", Completed, true},
		{"CANCELLED is valid", Cancelled, true},
		{"Invalid status", Status("INVALID"), false},
		{"Empty status", Status(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got, "%s.IsValid()", tt.status)
		})
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{Created, "CREATED"},
		{Processing, "PROCESSING"},
		{Held, "HELD"},
		{Released, "RELEASED"},
		{Completed, "COMPLETED"},
		{Cancelled, "CANCELLED"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			assert.Equal(t, tt.want, got)
		})
	}
}
