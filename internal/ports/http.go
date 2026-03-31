package ports

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/mcp-log/planning/internal/app/command"
	"github.com/mcp-log/planning/internal/app/query"
	"github.com/mcp-log/planning/internal/domain/plan"
	pkgerrors "github.com/mcp-log/planning/pkg/errors"
)

// HTTPHandler implements the planning HTTP API
type HTTPHandler struct {
	createPlan   *command.CreatePlanHandler
	addItem      *command.AddItemHandler
	removeItem   *command.RemoveItemHandler
	processPlan  *command.ProcessPlanHandler
	holdPlan     *command.HoldPlanHandler
	resumePlan   *command.ResumePlanHandler
	releasePlan  *command.ReleasePlanHandler
	completePlan *command.CompletePlanHandler
	cancelPlan   *command.CancelPlanHandler
	getPlan      *query.GetPlanHandler
	listPlans    *query.ListPlansHandler
	logger       *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler with all command/query handlers
func NewHTTPHandler(
	createPlan *command.CreatePlanHandler,
	addItem *command.AddItemHandler,
	removeItem *command.RemoveItemHandler,
	processPlan *command.ProcessPlanHandler,
	holdPlan *command.HoldPlanHandler,
	resumePlan *command.ResumePlanHandler,
	releasePlan *command.ReleasePlanHandler,
	completePlan *command.CompletePlanHandler,
	cancelPlan *command.CancelPlanHandler,
	getPlan *query.GetPlanHandler,
	listPlans *query.ListPlansHandler,
	logger *slog.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		createPlan:   createPlan,
		addItem:      addItem,
		removeItem:   removeItem,
		processPlan:  processPlan,
		holdPlan:     holdPlan,
		resumePlan:   resumePlan,
		releasePlan:  releasePlan,
		completePlan: completePlan,
		cancelPlan:   cancelPlan,
		getPlan:      getPlan,
		listPlans:    listPlans,
		logger:       logger,
	}
}

// --- Request/Response DTOs ---

type createPlanRequest struct {
	Name             string `json:"name"`
	Mode             string `json:"mode"`
	GroupingStrategy string `json:"groupingStrategy"`
	Priority         string `json:"priority"`
	MaxItems         int    `json:"maxItems"`
	Notes            string `json:"notes,omitempty"`
}

type addItemRequest struct {
	OrderID  string `json:"orderId"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type cancelPlanRequest struct {
	Reason string `json:"reason"`
}

type planItemDTO struct {
	ID       string    `json:"id"`
	OrderID  string    `json:"orderId"`
	SKU      string    `json:"sku"`
	Quantity int       `json:"quantity"`
	AddedAt  time.Time `json:"addedAt"`
}

type planResponseDTO struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Mode               string        `json:"mode"`
	GroupingStrategy   string        `json:"groupingStrategy"`
	Priority           string        `json:"priority"`
	Status             string        `json:"status"`
	MaxItems           int           `json:"maxItems"`
	Notes              string        `json:"notes,omitempty"`
	Items              []planItemDTO `json:"items"`
	CreatedAt          time.Time     `json:"createdAt"`
	UpdatedAt          time.Time     `json:"updatedAt"`
	ProcessedAt        *time.Time    `json:"processedAt,omitempty"`
	ReleasedAt         *time.Time    `json:"releasedAt,omitempty"`
	CompletedAt        *time.Time    `json:"completedAt,omitempty"`
	CancelledAt        *time.Time    `json:"cancelledAt,omitempty"`
	CancellationReason string        `json:"cancellationReason,omitempty"`
}

type planSummaryDTO struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Mode             string    `json:"mode"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority"`
	ItemCount        int       `json:"itemCount"`
	GroupingStrategy string    `json:"groupingStrategy"`
	CreatedAt        time.Time `json:"createdAt"`
}

type paginationDTO struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

type planListDTO struct {
	Data       []planSummaryDTO `json:"data"`
	Pagination paginationDTO    `json:"pagination"`
}

// --- Handlers ---

// HandleCreatePlan handles POST /v1/plans
func (h *HTTPHandler) HandleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeProblem(w, pkgerrors.NewValidationError("invalid request body: "+err.Error()))
		return
	}

	// Validate mode
	mode, err := plan.ParseMode(req.Mode)
	if err != nil {
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: "invalid mode: " + req.Mode,
		})
		return
	}

	// Validate grouping strategy
	strategy, err := plan.ParseGroupingStrategy(req.GroupingStrategy)
	if err != nil {
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: "invalid groupingStrategy: " + req.GroupingStrategy,
		})
		return
	}

	// Validate priority
	priority, err := plan.ParsePriority(req.Priority)
	if err != nil {
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: "invalid priority: " + req.Priority,
		})
		return
	}

	p, err := h.createPlan.Handle(r.Context(), req.Name, mode, strategy, priority, req.MaxItems, req.Notes)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toPlanResponse(p))
}

// HandleGetPlan handles GET /v1/plans/{planId}
func (h *HTTPHandler) HandleGetPlan(w http.ResponseWriter, r *http.Request, planID string) {
	p, err := h.getPlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// HandleListPlans handles GET /v1/plans
func (h *HTTPHandler) HandleListPlans(w http.ResponseWriter, r *http.Request) {
	q := query.ListPlansQuery{
		Cursor: r.URL.Query().Get("cursor"),
	}

	// Parse status filter
	if s := r.URL.Query().Get("status"); s != "" {
		status, err := plan.ParseStatus(s)
		if err != nil {
			h.writeProblem(w, pkgerrors.ProblemDetail{
				Type:   "https://problems.oms.io/validation-error",
				Title:  "Validation Error",
				Status: http.StatusUnprocessableEntity,
				Detail: "invalid status: " + s,
			})
			return
		}
		q.Status = &status
	}

	// Parse mode filter
	if m := r.URL.Query().Get("mode"); m != "" {
		mode, err := plan.ParseMode(m)
		if err != nil {
			h.writeProblem(w, pkgerrors.ProblemDetail{
				Type:   "https://problems.oms.io/validation-error",
				Title:  "Validation Error",
				Status: http.StatusUnprocessableEntity,
				Detail: "invalid mode: " + m,
			})
			return
		}
		q.Mode = &mode
	}

	// Parse priority filter
	if p := r.URL.Query().Get("priority"); p != "" {
		priority, err := plan.ParsePriority(p)
		if err != nil {
			h.writeProblem(w, pkgerrors.ProblemDetail{
				Type:   "https://problems.oms.io/validation-error",
				Title:  "Validation Error",
				Status: http.StatusUnprocessableEntity,
				Detail: "invalid priority: " + p,
			})
			return
		}
		q.Priority = &priority
	}

	// Parse limit
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil {
			h.writeProblem(w, pkgerrors.NewValidationError("invalid limit"))
			return
		}
		q.Limit = limit
	}

	result, err := h.listPlans.Handle(r.Context(), q)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	summaries := make([]planSummaryDTO, len(result.Plans))
	for i, p := range result.Plans {
		summaries[i] = planSummaryDTO{
			ID:               p.ID,
			Name:             p.Name,
			Mode:             string(p.Mode),
			Status:           string(p.Status),
			Priority:         string(p.Priority),
			ItemCount:        len(p.Items),
			GroupingStrategy: string(p.GroupingStrategy),
			CreatedAt:        p.CreatedAt,
		}
	}

	h.writeJSON(w, http.StatusOK, planListDTO{
		Data: summaries,
		Pagination: paginationDTO{
			NextCursor: result.NextCursor,
			HasMore:    result.HasMore,
		},
	})
}

// HandleGetPlanItems handles GET /v1/plans/{planId}/items
func (h *HTTPHandler) HandleGetPlanItems(w http.ResponseWriter, r *http.Request, planID string) {
	p, err := h.getPlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	items := make([]planItemDTO, len(p.Items))
	for i, item := range p.Items {
		items[i] = toPlanItemDTO(item)
	}
	h.writeJSON(w, http.StatusOK, items)
}

// HandleAddItem handles POST /v1/plans/{planId}/items
func (h *HTTPHandler) HandleAddItem(w http.ResponseWriter, r *http.Request, planID string) {
	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeProblem(w, pkgerrors.NewValidationError("invalid request body"))
		return
	}

	item, err := h.addItem.Handle(r.Context(), planID, req.OrderID, req.SKU, req.Quantity)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toPlanItemDTO(*item))
}

// HandleRemoveItem handles DELETE /v1/plans/{planId}/items/{itemId}
func (h *HTTPHandler) HandleRemoveItem(w http.ResponseWriter, r *http.Request, planID string, itemID string) {
	err := h.removeItem.Handle(r.Context(), planID, itemID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleProcessPlan handles POST /v1/plans/{planId}/process
func (h *HTTPHandler) HandleProcessPlan(w http.ResponseWriter, r *http.Request, planID string) {
	err := h.processPlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Fetch updated plan
	p, _ := h.getPlan.Handle(r.Context(), planID)
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// HandleHoldPlan handles POST /v1/plans/{planId}/hold
func (h *HTTPHandler) HandleHoldPlan(w http.ResponseWriter, r *http.Request, planID string) {
	err := h.holdPlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Fetch updated plan
	p, _ := h.getPlan.Handle(r.Context(), planID)
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// HandleResumePlan handles POST /v1/plans/{planId}/resume
func (h *HTTPHandler) HandleResumePlan(w http.ResponseWriter, r *http.Request, planID string) {
	err := h.resumePlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Fetch updated plan
	p, _ := h.getPlan.Handle(r.Context(), planID)
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// HandleReleasePlan handles POST /v1/plans/{planId}/release
func (h *HTTPHandler) HandleReleasePlan(w http.ResponseWriter, r *http.Request, planID string) {
	err := h.releasePlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Fetch updated plan
	p, _ := h.getPlan.Handle(r.Context(), planID)
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// HandleCompletePlan handles POST /v1/plans/{planId}/complete
func (h *HTTPHandler) HandleCompletePlan(w http.ResponseWriter, r *http.Request, planID string) {
	err := h.completePlan.Handle(r.Context(), planID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Fetch updated plan
	p, _ := h.getPlan.Handle(r.Context(), planID)
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// HandleCancelPlan handles POST /v1/plans/{planId}/cancel
func (h *HTTPHandler) HandleCancelPlan(w http.ResponseWriter, r *http.Request, planID string) {
	var req cancelPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeProblem(w, pkgerrors.NewValidationError("invalid request body"))
		return
	}

	err := h.cancelPlan.Handle(r.Context(), planID, req.Reason)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Fetch updated plan
	p, _ := h.getPlan.Handle(r.Context(), planID)
	h.writeJSON(w, http.StatusOK, toPlanResponse(p))
}

// --- Helpers ---

func (h *HTTPHandler) handleDomainError(w http.ResponseWriter, err error) {
	var transErr plan.ErrInvalidTransition
	switch {
	case errors.Is(err, plan.ErrPlanNotFound):
		h.writeProblem(w, pkgerrors.NewNotFoundError("Plan", ""))
	case errors.Is(err, plan.ErrItemNotFound):
		h.writeProblem(w, pkgerrors.NewNotFoundError("PlanItem", ""))
	case errors.As(err, &transErr):
		h.writeProblem(w, pkgerrors.NewConflictError(err.Error()))
	case errors.Is(err, plan.ErrDuplicateItem):
		h.writeProblem(w, pkgerrors.NewConflictError(err.Error()))
	case errors.Is(err, plan.ErrItemsNotAllowed):
		h.writeProblem(w, pkgerrors.NewConflictError(err.Error()))
	case errors.Is(err, plan.ErrPlanFull):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	case errors.Is(err, plan.ErrEmptyPlan):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	case errors.Is(err, plan.ErrInvalidName):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	case errors.Is(err, plan.ErrCancelReasonRequired):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	case errors.Is(err, plan.ErrInvalidQuantity):
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/validation-error",
			Title:  "Validation Error",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
	default:
		h.logger.Error("unhandled error", "error", err)
		h.writeProblem(w, pkgerrors.ProblemDetail{
			Type:   "https://problems.oms.io/internal-error",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "an unexpected error occurred",
		})
	}
}

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *HTTPHandler) writeProblem(w http.ResponseWriter, problem pkgerrors.ProblemDetail) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		h.logger.Error("failed to encode problem response", "error", err)
	}
}

func toPlanResponse(p *plan.Plan) planResponseDTO {
	items := make([]planItemDTO, len(p.Items))
	for i, item := range p.Items {
		items[i] = toPlanItemDTO(item)
	}
	return planResponseDTO{
		ID:                 p.ID,
		Name:               p.Name,
		Mode:               string(p.Mode),
		GroupingStrategy:   string(p.GroupingStrategy),
		Priority:           string(p.Priority),
		Status:             string(p.Status),
		MaxItems:           p.MaxItems,
		Notes:              p.Notes,
		Items:              items,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		ProcessedAt:        p.ProcessedAt,
		ReleasedAt:         p.ReleasedAt,
		CompletedAt:        p.CompletedAt,
		CancelledAt:        p.CancelledAt,
		CancellationReason: p.CancellationReason,
	}
}

func toPlanItemDTO(item plan.PlanItem) planItemDTO {
	return planItemDTO{
		ID:       item.ID,
		OrderID:  item.OrderID,
		SKU:      item.SKU,
		Quantity: item.Quantity,
		AddedAt:  item.AddedAt,
	}
}
