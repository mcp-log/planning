package plan

import (
	"fmt"
	"strings"
	"time"

	"github.com/mcp-log/planning/pkg/events"
	"github.com/mcp-log/planning/pkg/identity"
)

// Mode represents the planning strategy
type Mode string

const (
	// Wave mode: batch-based processing (accumulate, process, release)
	Wave Mode = "WAVE"

	// Dynamic mode: continuous streaming (auto-release on first item)
	Dynamic Mode = "DYNAMIC"
)

// Strategy represents the grouping criteria for batching orders
type Strategy string

const (
	StrategyCarrier  Strategy = "CARRIER"
	StrategyZone     Strategy = "ZONE"
	StrategyPriority Strategy = "PRIORITY"
	StrategyChannel  Strategy = "CHANNEL"
	StrategyNone     Strategy = "NONE"
)

// Priority represents the urgency level for plan execution
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityNormal Priority = "NORMAL"
	PriorityHigh   Priority = "HIGH"
	PriorityRush   Priority = "RUSH"
)

// PlanItem represents a reference to an order line item assigned to a plan
type PlanItem struct {
	ID       string
	PlanID   string
	OrderID  string
	SKU      string
	Quantity int
	AddedAt  time.Time
}

// Plan is the aggregate root for fulfillment planning
type Plan struct {
	ID                  string
	Name                string
	Mode                Mode
	GroupingStrategy    Strategy
	Priority            Priority
	Status              Status
	MaxItems            int
	Notes               string
	Items               []PlanItem
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ProcessedAt         *time.Time
	ReleasedAt          *time.Time
	CompletedAt         *time.Time
	CancelledAt         *time.Time
	CancellationReason  string
	domainEvents        []events.DomainEvent
}

// NewPlan creates a new Plan aggregate
func NewPlan(name string, mode Mode, groupingStrategy Strategy, priority Priority, maxItems int, notes string) (*Plan, error) {
	// Validate name
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidName
	}

	// Validate maxItems
	if maxItems < 0 {
		return nil, fmt.Errorf("maxItems cannot be negative")
	}

	now := time.Now()
	plan := &Plan{
		ID:               identity.NewID(),
		Name:             name,
		Mode:             mode,
		GroupingStrategy: groupingStrategy,
		Priority:         priority,
		Status:           Created,
		MaxItems:         maxItems,
		Notes:            notes,
		Items:            []PlanItem{},
		CreatedAt:        now,
		UpdatedAt:        now,
		domainEvents:     []events.DomainEvent{},
	}

	plan.addEvent(NewPlanCreatedEvent(plan))
	return plan, nil
}

// AddItem adds an order item reference to the plan
func (p *Plan) AddItem(orderID string, sku string, quantity int) error {
	// Validate quantity
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Check if items can be modified
	// DYNAMIC plans can add items even when RELEASED (continuous streaming)
	if p.Status == Completed || p.Status == Cancelled {
		return ErrItemsNotAllowed
	}
	if p.Status == Released && p.Mode == Wave {
		return ErrItemsNotAllowed
	}

	// Check capacity (0 means unlimited)
	if p.MaxItems > 0 && len(p.Items) >= p.MaxItems {
		return ErrPlanFull
	}

	// Check for duplicate (orderID, sku)
	for _, item := range p.Items {
		if item.OrderID == orderID && item.SKU == sku {
			return ErrDuplicateItem
		}
	}

	// Create new item
	item := PlanItem{
		ID:       identity.NewID(),
		PlanID:   p.ID,
		OrderID:  orderID,
		SKU:      sku,
		Quantity: quantity,
		AddedAt:  time.Now(),
	}

	p.Items = append(p.Items, item)
	p.addEvent(NewPlanItemAddedEvent(p, item))

	// Auto-release for DYNAMIC mode on first item
	if p.Mode == Dynamic && len(p.Items) == 1 {
		now := time.Now()
		p.ReleasedAt = &now
		p.addEvent(NewPlanReleasedEvent(p))
		p.transitionTo(Released)
	}

	return nil
}

// RemoveItem removes an item from the plan
func (p *Plan) RemoveItem(itemID string) error {
	// Check if items can be modified
	if p.Status == Released || p.Status == Completed || p.Status == Cancelled {
		return ErrItemsNotAllowed
	}

	// Find and remove item
	for i, item := range p.Items {
		if item.ID == itemID {
			// Remove item
			removedItem := item
			p.Items = append(p.Items[:i], p.Items[i+1:]...)
			p.addEvent(NewPlanItemRemovedEvent(p, removedItem))
			return nil
		}
	}

	return ErrItemNotFound
}

// Process transitions the plan to PROCESSING status (WAVE mode only)
func (p *Plan) Process() error {
	// DYNAMIC plans cannot be processed
	if p.Mode == Dynamic {
		return NewErrInvalidTransition(p.Status, Processing, "DYNAMIC plans cannot be processed")
	}

	// Check if plan has items
	if len(p.Items) == 0 {
		return ErrEmptyPlan
	}

	// Check valid transition
	if !CanTransitionTo(p.Status, Processing) {
		return NewErrInvalidTransition(p.Status, Processing, "")
	}

	now := time.Now()
	p.ProcessedAt = &now
	p.addEvent(NewPlanProcessedEvent(p))
	p.transitionTo(Processing)

	return nil
}

// Hold pauses processing of the plan (PROCESSING → HELD)
func (p *Plan) Hold() error {
	if !CanTransitionTo(p.Status, Held) {
		return NewErrInvalidTransition(p.Status, Held, "")
	}

	now := time.Now()
	p.addEvent(NewPlanHeldEvent(p, now))
	p.transitionTo(Held)

	return nil
}

// Resume resumes processing of the plan (HELD → PROCESSING)
func (p *Plan) Resume() error {
	if !CanTransitionTo(p.Status, Processing) {
		return NewErrInvalidTransition(p.Status, Processing, "")
	}

	now := time.Now()
	p.addEvent(NewPlanResumedEvent(p, now))
	p.transitionTo(Processing)

	return nil
}

// Release releases the plan to the warehouse floor
func (p *Plan) Release() error {
	// WAVE plans cannot be released directly from CREATED (must process first)
	if p.Mode == Wave && p.Status == Created {
		return NewErrInvalidTransition(p.Status, Released, "must process first for WAVE mode")
	}

	if !CanTransitionTo(p.Status, Released) {
		return NewErrInvalidTransition(p.Status, Released, "")
	}

	now := time.Now()
	p.ReleasedAt = &now
	p.addEvent(NewPlanReleasedEvent(p))
	p.transitionTo(Released)

	return nil
}

// Complete marks the plan as completed
func (p *Plan) Complete() error {
	if !CanTransitionTo(p.Status, Completed) {
		return NewErrInvalidTransition(p.Status, Completed, "")
	}

	now := time.Now()
	p.CompletedAt = &now
	p.addEvent(NewPlanCompletedEvent(p))
	p.transitionTo(Completed)

	return nil
}

// Cancel cancels the plan with a reason
func (p *Plan) Cancel(reason string) error {
	// Validate reason
	if strings.TrimSpace(reason) == "" {
		return ErrCancelReasonRequired
	}

	if !CanTransitionTo(p.Status, Cancelled) {
		return NewErrInvalidTransition(p.Status, Cancelled, "")
	}

	previousStatus := p.Status
	now := time.Now()
	p.CancelledAt = &now
	p.CancellationReason = reason
	p.addEvent(NewPlanCancelledEvent(p, previousStatus))
	p.transitionTo(Cancelled)

	return nil
}

// transitionTo transitions the plan to a new status and emits status changed event
func (p *Plan) transitionTo(newStatus Status) {
	oldStatus := p.Status
	p.Status = newStatus
	p.UpdatedAt = time.Now()
	p.addEvent(NewPlanStatusChangedEvent(p, oldStatus, newStatus))
}

// DomainEvents returns collected domain events
func (p *Plan) DomainEvents() []events.DomainEvent {
	return p.domainEvents
}

// ClearEvents clears collected domain events
func (p *Plan) ClearEvents() {
	p.domainEvents = nil
}

// addEvent adds a domain event to the collection
func (p *Plan) addEvent(event events.DomainEvent) {
	p.domainEvents = append(p.domainEvents, event)
}

// --- Parse Functions ---

// ParseMode parses a string into a Mode
func ParseMode(s string) (Mode, error) {
	mode := Mode(strings.ToUpper(s))
	switch mode {
	case Wave, Dynamic:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid mode: %s", s)
	}
}

// ParseGroupingStrategy parses a string into a Strategy
func ParseGroupingStrategy(s string) (Strategy, error) {
	strategy := Strategy(strings.ToUpper(s))
	switch strategy {
	case StrategyCarrier, StrategyZone, StrategyPriority, StrategyChannel, StrategyNone:
		return strategy, nil
	default:
		return "", fmt.Errorf("invalid grouping strategy: %s", s)
	}
}

// ParsePriority parses a string into a Priority
func ParsePriority(s string) (Priority, error) {
	priority := Priority(strings.ToUpper(s))
	switch priority {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityRush:
		return priority, nil
	default:
		return "", fmt.Errorf("invalid priority: %s", s)
	}
}

// ParseStatus parses a string into a Status
func ParseStatus(s string) (Status, error) {
	status := Status(strings.ToUpper(s))
	if !status.IsValid() {
		return "", fmt.Errorf("invalid status: %s", s)
	}
	return status, nil
}
