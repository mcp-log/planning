package plan

import (
	"testing"

	"github.com/mcp-log/planning/pkg/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Constructor Tests

func TestNewPlan_ValidCreation(t *testing.T) {
	name := "Test Plan"
	mode := Wave
	strategy := StrategyCarrier
	priority := PriorityNormal
	maxItems := 100
	notes := "Test notes"

	plan, err := NewPlan(name, mode, strategy, priority, maxItems, notes)

	require.NoError(t, err)
	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, name, plan.Name)
	assert.Equal(t, mode, plan.Mode)
	assert.Equal(t, strategy, plan.GroupingStrategy)
	assert.Equal(t, priority, plan.Priority)
	assert.Equal(t, Created, plan.Status)
	assert.Equal(t, maxItems, plan.MaxItems)
	assert.Equal(t, notes, plan.Notes)
	assert.NotZero(t, plan.CreatedAt)
	assert.NotZero(t, plan.UpdatedAt)
	assert.Empty(t, plan.Items)

	// Check event
	events := plan.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePlanCreated, events[0].EventType())
}

func TestNewPlan_EmptyName(t *testing.T) {
	tests := []struct {
		name     string
		planName string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs only", "\t\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlan(tt.planName, Wave, StrategyNone, PriorityNormal, 0, "")
			assert.ErrorIs(t, err, ErrInvalidName)
		})
	}
}

func TestNewPlan_NegativeMaxItems(t *testing.T) {
	_, err := NewPlan("Test", Wave, StrategyNone, PriorityNormal, -1, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maxItems cannot be negative")
}

func TestNewPlan_DefaultValues(t *testing.T) {
	plan, err := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")

	require.NoError(t, err)
	assert.Equal(t, StrategyNone, plan.GroupingStrategy)
	assert.Equal(t, PriorityNormal, plan.Priority)
	assert.Equal(t, Created, plan.Status)
	assert.Equal(t, 0, plan.MaxItems) // 0 means unlimited
}

// AddItem Tests

func TestAddItem_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	orderID := identity.NewID()
	sku := "WIDGET-001"
	quantity := 5

	err := plan.AddItem(orderID, sku, quantity)

	require.NoError(t, err)
	assert.Len(t, plan.Items, 1)
	assert.Equal(t, orderID, plan.Items[0].OrderID)
	assert.Equal(t, sku, plan.Items[0].SKU)
	assert.Equal(t, quantity, plan.Items[0].Quantity)
	assert.NotEmpty(t, plan.Items[0].ID)

	// Check event
	events := plan.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePlanItemAdded, events[0].EventType())
}

func TestAddItem_ExceedsCapacity(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 2, "")
	plan.ClearEvents()

	// Add two items (at capacity)
	_ = plan.AddItem(identity.NewID(), "SKU-1", 1)
	_ = plan.AddItem(identity.NewID(), "SKU-2", 1)

	// Try to add third item (should fail)
	err := plan.AddItem(identity.NewID(), "SKU-3", 1)
	assert.ErrorIs(t, err, ErrPlanFull)
	assert.Len(t, plan.Items, 2) // Still only 2 items
}

func TestAddItem_DuplicateOrderAndSKU(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	orderID := identity.NewID()
	sku := "WIDGET-001"

	// Add item first time
	err := plan.AddItem(orderID, sku, 5)
	require.NoError(t, err)

	// Try to add same (orderID, sku) again
	err = plan.AddItem(orderID, sku, 10)
	assert.ErrorIs(t, err, ErrDuplicateItem)
	assert.Len(t, plan.Items, 1) // Still only 1 item
}

func TestAddItem_InvalidQuantity(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	tests := []struct {
		name     string
		quantity int
	}{
		{"zero quantity", 0},
		{"negative quantity", -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plan.AddItem(identity.NewID(), "SKU", tt.quantity)
			assert.ErrorIs(t, err, ErrInvalidQuantity)
		})
	}
}

func TestAddItem_InvalidStatus(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.Status = Completed
	plan.ClearEvents()

	err := plan.AddItem(identity.NewID(), "SKU", 1)
	assert.ErrorIs(t, err, ErrItemsNotAllowed)
}

func TestAddItem_DynamicPlanAutoRelease(t *testing.T) {
	plan, _ := NewPlan("Test", Dynamic, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	assert.Equal(t, Created, plan.Status)

	// Add first item to DYNAMIC plan
	err := plan.AddItem(identity.NewID(), "SKU", 1)
	require.NoError(t, err)

	// Plan should auto-transition to RELEASED
	assert.Equal(t, Released, plan.Status)
	assert.NotNil(t, plan.ReleasedAt)

	// Check events: PlanItemAdded + PlanReleased + PlanStatusChanged
	events := plan.DomainEvents()
	require.Len(t, events, 3)
	assert.Equal(t, EventTypePlanItemAdded, events[0].EventType())
	assert.Equal(t, EventTypePlanReleased, events[1].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[2].EventType())
}

func TestAddItem_WavePlanNoAutoRelease(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	assert.Equal(t, Created, plan.Status)

	// Add first item to WAVE plan
	err := plan.AddItem(identity.NewID(), "SKU", 1)
	require.NoError(t, err)

	// Plan should remain in CREATED status (no auto-release for WAVE)
	assert.Equal(t, Created, plan.Status)
	assert.Nil(t, plan.ReleasedAt)

	// Check events: only PlanItemAdded
	events := plan.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePlanItemAdded, events[0].EventType())
}

// RemoveItem Tests

func TestRemoveItem_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	orderID := identity.NewID()
	_ = plan.AddItem(orderID, "SKU", 1)
	itemID := plan.Items[0].ID
	plan.ClearEvents()

	err := plan.RemoveItem(itemID)

	require.NoError(t, err)
	assert.Empty(t, plan.Items)

	// Check event
	events := plan.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePlanItemRemoved, events[0].EventType())
}

func TestRemoveItem_NotFound(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	err := plan.RemoveItem(identity.NewID())
	assert.ErrorIs(t, err, ErrItemNotFound)
}

func TestRemoveItem_InvalidStatus(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	itemID := plan.Items[0].ID

	plan.Status = Released
	plan.ClearEvents()

	err := plan.RemoveItem(itemID)
	assert.ErrorIs(t, err, ErrItemsNotAllowed)
}

// Process Tests

func TestProcess_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	plan.ClearEvents()

	err := plan.Process()

	require.NoError(t, err)
	assert.Equal(t, Processing, plan.Status)
	assert.NotNil(t, plan.ProcessedAt)

	// Check events: PlanProcessed + PlanStatusChanged
	events := plan.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, EventTypePlanProcessed, events[0].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[1].EventType())
}

func TestProcess_EmptyPlan(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	err := plan.Process()
	assert.ErrorIs(t, err, ErrEmptyPlan)
	assert.Equal(t, Created, plan.Status)
}

func TestProcess_DynamicPlan(t *testing.T) {
	plan, _ := NewPlan("Test", Dynamic, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	err := plan.Process()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DYNAMIC plans cannot be processed")
}

func TestProcess_InvalidTransition(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	plan.Status = Released
	plan.ClearEvents()

	err := plan.Process()
	var transErr ErrInvalidTransition
	assert.ErrorAs(t, err, &transErr)
	assert.Equal(t, Released, transErr.From)
	assert.Equal(t, Processing, transErr.To)
}

// Hold and Resume Tests

func TestHold_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	_ = plan.Process()
	plan.ClearEvents()

	err := plan.Hold()

	require.NoError(t, err)
	assert.Equal(t, Held, plan.Status)

	// Check events
	events := plan.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, EventTypePlanHeld, events[0].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[1].EventType())
}

func TestResume_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	_ = plan.Process()
	_ = plan.Hold()
	plan.ClearEvents()

	err := plan.Resume()

	require.NoError(t, err)
	assert.Equal(t, Processing, plan.Status)

	// Check events
	events := plan.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, EventTypePlanResumed, events[0].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[1].EventType())
}

// Release Tests

func TestRelease_WaveSuccess(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	_ = plan.Process()
	plan.ClearEvents()

	err := plan.Release()

	require.NoError(t, err)
	assert.Equal(t, Released, plan.Status)
	assert.NotNil(t, plan.ReleasedAt)

	// Check events
	events := plan.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, EventTypePlanReleased, events[0].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[1].EventType())
}

func TestRelease_WaveFromCreatedFails(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	plan.ClearEvents()

	err := plan.Release()
	var transErr ErrInvalidTransition
	assert.ErrorAs(t, err, &transErr)
	assert.Equal(t, Created, transErr.From)
	assert.Equal(t, Released, transErr.To)
}

// Complete Tests

func TestComplete_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	_ = plan.Process()
	_ = plan.Release()
	plan.ClearEvents()

	err := plan.Complete()

	require.NoError(t, err)
	assert.Equal(t, Completed, plan.Status)
	assert.NotNil(t, plan.CompletedAt)

	// Check events
	events := plan.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, EventTypePlanCompleted, events[0].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[1].EventType())
}

// Cancel Tests

func TestCancel_Success(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	reason := "Test cancellation reason"
	err := plan.Cancel(reason)

	require.NoError(t, err)
	assert.Equal(t, Cancelled, plan.Status)
	assert.NotNil(t, plan.CancelledAt)
	assert.Equal(t, reason, plan.CancellationReason)

	// Check events
	events := plan.DomainEvents()
	require.Len(t, events, 2)
	assert.Equal(t, EventTypePlanCancelled, events[0].EventType())
	assert.Equal(t, EventTypePlanStatusChanged, events[1].EventType())
}

func TestCancel_EmptyReason(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	tests := []string{"", "   ", "\t\t"}
	for _, reason := range tests {
		err := plan.Cancel(reason)
		assert.ErrorIs(t, err, ErrCancelReasonRequired)
	}
}

func TestCancel_TerminalStateFails(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	_ = plan.AddItem(identity.NewID(), "SKU", 1)
	_ = plan.Process()
	_ = plan.Release()
	_ = plan.Complete()
	plan.ClearEvents()

	err := plan.Cancel("Too late")
	var transErr ErrInvalidTransition
	assert.ErrorAs(t, err, &transErr)
	assert.Equal(t, Completed, transErr.From)
	assert.Equal(t, Cancelled, transErr.To)
}

// Event Collection Tests

func TestDomainEvents_Collection(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	plan.ClearEvents()

	_ = plan.AddItem(identity.NewID(), "SKU-1", 1)
	_ = plan.AddItem(identity.NewID(), "SKU-2", 2)

	events := plan.DomainEvents()
	assert.Len(t, events, 2)
	assert.Equal(t, EventTypePlanItemAdded, events[0].EventType())
	assert.Equal(t, EventTypePlanItemAdded, events[1].EventType())
}

func TestClearEvents(t *testing.T) {
	plan, _ := NewPlan("Test", Wave, StrategyNone, PriorityNormal, 0, "")
	assert.Len(t, plan.DomainEvents(), 1) // PlanCreated

	plan.ClearEvents()
	assert.Empty(t, plan.DomainEvents())
}

// Full Lifecycle Tests

func TestWaveLifecycle(t *testing.T) {
	// CREATED → AddItems → PROCESSING → RELEASED → COMPLETED
	plan, err := NewPlan("Wave Test", Wave, StrategyCarrier, PriorityHigh, 10, "Full lifecycle test")
	require.NoError(t, err)
	assert.Equal(t, Created, plan.Status)

	// Add items
	for i := 0; i < 3; i++ {
		err := plan.AddItem(identity.NewID(), "SKU-"+string(rune('A'+i)), i+1)
		require.NoError(t, err)
	}
	assert.Len(t, plan.Items, 3)
	assert.Equal(t, Created, plan.Status) // Still CREATED

	// Process
	err = plan.Process()
	require.NoError(t, err)
	assert.Equal(t, Processing, plan.Status)
	assert.NotNil(t, plan.ProcessedAt)

	// Release
	err = plan.Release()
	require.NoError(t, err)
	assert.Equal(t, Released, plan.Status)
	assert.NotNil(t, plan.ReleasedAt)

	// Complete
	err = plan.Complete()
	require.NoError(t, err)
	assert.Equal(t, Completed, plan.Status)
	assert.NotNil(t, plan.CompletedAt)
	assert.True(t, plan.Status.IsTerminal())
}

func TestDynamicLifecycle(t *testing.T) {
	// CREATED → AddItem (auto-release) → RELEASED → COMPLETED
	plan, err := NewPlan("Dynamic Test", Dynamic, StrategyNone, PriorityNormal, 0, "Dynamic lifecycle test")
	require.NoError(t, err)
	assert.Equal(t, Created, plan.Status)

	// Add first item (triggers auto-release)
	err = plan.AddItem(identity.NewID(), "SKU-1", 1)
	require.NoError(t, err)
	assert.Equal(t, Released, plan.Status) // Auto-released!
	assert.NotNil(t, plan.ReleasedAt)

	// Add more items (plan already RELEASED, continuous streaming)
	err = plan.AddItem(identity.NewID(), "SKU-2", 2)
	require.NoError(t, err)
	assert.Len(t, plan.Items, 2)
	assert.Equal(t, Released, plan.Status)

	// Complete
	err = plan.Complete()
	require.NoError(t, err)
	assert.Equal(t, Completed, plan.Status)
	assert.True(t, plan.Status.IsTerminal())
}
