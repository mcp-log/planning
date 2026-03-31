package plan

import (
	"time"

	"github.com/mcp-log/planning/pkg/events"
)

// Event types
const (
	EventTypePlanCreated       = "plan.created"
	EventTypePlanItemAdded     = "plan.item_added"
	EventTypePlanItemRemoved   = "plan.item_removed"
	EventTypePlanProcessed     = "plan.processed"
	EventTypePlanHeld          = "plan.held"
	EventTypePlanResumed       = "plan.resumed"
	EventTypePlanReleased      = "plan.released"
	EventTypePlanCompleted     = "plan.completed"
	EventTypePlanCancelled     = "plan.cancelled"
	EventTypePlanStatusChanged = "plan.status_changed"
)

// PlanCreated event
type PlanCreated struct {
	events.BaseEvent
	PlanID           string `json:"planId"`
	Name             string    `json:"name"`
	Mode             Mode      `json:"mode"`
	GroupingStrategy Strategy  `json:"groupingStrategy"`
	Priority         Priority  `json:"priority"`
	Status           Status    `json:"status"`
	MaxItems         int       `json:"maxItems"`
	Notes            string    `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

// NewPlanCreatedEvent creates a new PlanCreated event
func NewPlanCreatedEvent(p *Plan) *PlanCreated {
	return &PlanCreated{
		BaseEvent:        events.NewBaseEvent(EventTypePlanCreated, p.ID, "plan"),
		PlanID:           p.ID,
		Name:             p.Name,
		Mode:             p.Mode,
		GroupingStrategy: p.GroupingStrategy,
		Priority:         p.Priority,
		Status:           p.Status,
		MaxItems:         p.MaxItems,
		Notes:            p.Notes,
		CreatedAt:        p.CreatedAt,
	}
}

// PlanItemAdded event
type PlanItemAdded struct {
	events.BaseEvent
	PlanID           string `json:"planId"`
	ItemID           string `json:"itemId"`
	OrderID          string `json:"orderId"`
	SKU              string    `json:"sku"`
	Quantity         int       `json:"quantity"`
	AddedAt          time.Time `json:"addedAt"`
	CurrentItemCount int       `json:"currentItemCount"`
}

// NewPlanItemAddedEvent creates a new PlanItemAdded event
func NewPlanItemAddedEvent(p *Plan, item PlanItem) *PlanItemAdded {
	return &PlanItemAdded{
		BaseEvent:        events.NewBaseEvent(EventTypePlanItemAdded, p.ID, "plan"),
		PlanID:           p.ID,
		ItemID:           item.ID,
		OrderID:          item.OrderID,
		SKU:              item.SKU,
		Quantity:         item.Quantity,
		AddedAt:          item.AddedAt,
		CurrentItemCount: len(p.Items),
	}
}

// PlanItemRemoved event
type PlanItemRemoved struct {
	events.BaseEvent
	PlanID           string `json:"planId"`
	ItemID           string `json:"itemId"`
	OrderID          string `json:"orderId"`
	SKU              string    `json:"sku"`
	CurrentItemCount int       `json:"currentItemCount"`
}

// NewPlanItemRemovedEvent creates a new PlanItemRemoved event
func NewPlanItemRemovedEvent(p *Plan, item PlanItem) *PlanItemRemoved {
	return &PlanItemRemoved{
		BaseEvent:        events.NewBaseEvent(EventTypePlanItemRemoved, p.ID, "plan"),
		PlanID:           p.ID,
		ItemID:           item.ID,
		OrderID:          item.OrderID,
		SKU:              item.SKU,
		CurrentItemCount: len(p.Items),
	}
}

// PlanProcessed event
type PlanProcessed struct {
	events.BaseEvent
	PlanID      string `json:"planId"`
	ProcessedAt time.Time `json:"processedAt"`
	ItemCount   int       `json:"itemCount"`
}

// NewPlanProcessedEvent creates a new PlanProcessed event
func NewPlanProcessedEvent(p *Plan) *PlanProcessed {
	return &PlanProcessed{
		BaseEvent:   events.NewBaseEvent(EventTypePlanProcessed, p.ID, "plan"),
		PlanID:      p.ID,
		ProcessedAt: *p.ProcessedAt,
		ItemCount:   len(p.Items),
	}
}

// PlanHeld event
type PlanHeld struct {
	events.BaseEvent
	PlanID string `json:"planId"`
	HeldAt time.Time `json:"heldAt"`
}

// NewPlanHeldEvent creates a new PlanHeld event
func NewPlanHeldEvent(p *Plan, heldAt time.Time) *PlanHeld {
	return &PlanHeld{
		BaseEvent: events.NewBaseEvent(EventTypePlanHeld, p.ID, "plan"),
		PlanID:    p.ID,
		HeldAt:    heldAt,
	}
}

// PlanResumed event
type PlanResumed struct {
	events.BaseEvent
	PlanID    string `json:"planId"`
	ResumedAt time.Time `json:"resumedAt"`
}

// NewPlanResumedEvent creates a new PlanResumed event
func NewPlanResumedEvent(p *Plan, resumedAt time.Time) *PlanResumed {
	return &PlanResumed{
		BaseEvent: events.NewBaseEvent(EventTypePlanResumed, p.ID, "plan"),
		PlanID:    p.ID,
		ResumedAt: resumedAt,
	}
}

// PlanReleased event
type PlanReleased struct {
	events.BaseEvent
	PlanID     string `json:"planId"`
	ReleasedAt time.Time `json:"releasedAt"`
	ItemCount  int       `json:"itemCount"`
	Mode       Mode      `json:"mode"`
}

// NewPlanReleasedEvent creates a new PlanReleased event
func NewPlanReleasedEvent(p *Plan) *PlanReleased {
	return &PlanReleased{
		BaseEvent:  events.NewBaseEvent(EventTypePlanReleased, p.ID, "plan"),
		PlanID:     p.ID,
		ReleasedAt: *p.ReleasedAt,
		ItemCount:  len(p.Items),
		Mode:       p.Mode,
	}
}

// PlanCompleted event
type PlanCompleted struct {
	events.BaseEvent
	PlanID      string `json:"planId"`
	CompletedAt time.Time `json:"completedAt"`
	ItemCount   int       `json:"itemCount"`
}

// NewPlanCompletedEvent creates a new PlanCompleted event
func NewPlanCompletedEvent(p *Plan) *PlanCompleted {
	return &PlanCompleted{
		BaseEvent:   events.NewBaseEvent(EventTypePlanCompleted, p.ID, "plan"),
		PlanID:      p.ID,
		CompletedAt: *p.CompletedAt,
		ItemCount:   len(p.Items),
	}
}

// PlanCancelled event
type PlanCancelled struct {
	events.BaseEvent
	PlanID         string `json:"planId"`
	CancelledAt    time.Time `json:"cancelledAt"`
	Reason         string    `json:"reason"`
	PreviousStatus Status    `json:"previousStatus"`
}

// NewPlanCancelledEvent creates a new PlanCancelled event
func NewPlanCancelledEvent(p *Plan, previousStatus Status) *PlanCancelled {
	return &PlanCancelled{
		BaseEvent:      events.NewBaseEvent(EventTypePlanCancelled, p.ID, "plan"),
		PlanID:         p.ID,
		CancelledAt:    *p.CancelledAt,
		Reason:         p.CancellationReason,
		PreviousStatus: previousStatus,
	}
}

// PlanStatusChanged event
type PlanStatusChanged struct {
	events.BaseEvent
	PlanID    string `json:"planId"`
	OldStatus Status    `json:"oldStatus"`
	NewStatus Status    `json:"newStatus"`
	ChangedAt time.Time `json:"changedAt"`
}

// NewPlanStatusChangedEvent creates a new PlanStatusChanged event
func NewPlanStatusChangedEvent(p *Plan, oldStatus, newStatus Status) *PlanStatusChanged {
	return &PlanStatusChanged{
		BaseEvent: events.NewBaseEvent(EventTypePlanStatusChanged, p.ID, "plan"),
		PlanID:    p.ID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedAt: time.Now(),
	}
}
