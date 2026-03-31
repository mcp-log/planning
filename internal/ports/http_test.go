package ports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/mcp-log/planning/internal/app/command"
	"github.com/mcp-log/planning/internal/app/query"
	"github.com/mcp-log/planning/internal/domain/plan"
	"github.com/mcp-log/planning/internal/ports"
	"github.com/mcp-log/planning/pkg/events"
	"github.com/mcp-log/planning/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- In-Memory Repository for Testing ---

type inMemoryRepo struct {
	mu    sync.RWMutex
	plans map[string]*plan.Plan
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		plans: make(map[string]*plan.Plan),
	}
}

func (r *inMemoryRepo) Save(_ context.Context, p *plan.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[p.ID] = p
	return nil
}

func (r *inMemoryRepo) FindByID(_ context.Context, id string) (*plan.Plan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plans[id]
	if !ok {
		return nil, plan.ErrPlanNotFound
	}
	return p, nil
}

func (r *inMemoryRepo) Update(_ context.Context, p *plan.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[p.ID] = p
	return nil
}

func (r *inMemoryRepo) List(_ context.Context, filter plan.ListFilter, limit int, cursor string) ([]*plan.Plan, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*plan.Plan
	for _, p := range r.plans {
		if filter.Status != nil && p.Status != *filter.Status {
			continue
		}
		if filter.Mode != nil && p.Mode != *filter.Mode {
			continue
		}
		if filter.Priority != nil && p.Priority != *filter.Priority {
			continue
		}
		result = append(result, p)
	}

	if limit <= 0 {
		limit = pagination.DefaultLimit
	}

	if len(result) > limit {
		result = result[:limit]
	}
	return result, "", nil
}

// --- Test Event Publisher ---

type testPublisher struct {
	mu     sync.Mutex
	events []events.DomainEvent
}

func (p *testPublisher) Publish(_ context.Context, evt events.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, evt)
	return nil
}

func (p *testPublisher) PublishBatch(_ context.Context, evts []events.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, evts...)
	return nil
}

func (p *testPublisher) Events() []events.DomainEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]events.DomainEvent{}, p.events...)
}

func (p *testPublisher) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = nil
}

// --- Test Setup ---

func setupTestServer(t *testing.T) (*httptest.Server, *inMemoryRepo, *testPublisher) {
	t.Helper()

	repo := newInMemoryRepo()
	pub := &testPublisher{}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	handler := ports.NewHTTPHandler(
		command.NewCreatePlanHandler(repo, pub),
		command.NewAddItemHandler(repo, pub),
		command.NewRemoveItemHandler(repo, pub),
		command.NewProcessPlanHandler(repo, pub),
		command.NewHoldPlanHandler(repo, pub),
		command.NewResumePlanHandler(repo, pub),
		command.NewReleasePlanHandler(repo, pub),
		command.NewCompletePlanHandler(repo, pub),
		command.NewCancelPlanHandler(repo, pub),
		query.NewGetPlanHandler(repo),
		query.NewListPlansHandler(repo),
		logger,
	)

	router := ports.NewRouter(handler)
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)

	return ts, repo, pub
}

func createPlanPayload(mode string) map[string]interface{} {
	return map[string]interface{}{
		"name":             "Test Plan",
		"mode":             mode,
		"groupingStrategy": "CARRIER",
		"priority":         "NORMAL",
		"maxItems":         100,
		"notes":            "Test plan notes",
	}
}

func addItemPayload(orderID, sku string, quantity int) map[string]interface{} {
	return map[string]interface{}{
		"orderId":  orderID,
		"sku":      sku,
		"quantity": quantity,
	}
}

func cancelPlanPayload(reason string) map[string]interface{} {
	return map[string]interface{}{
		"reason": reason,
	}
}

func postJSON(ts *httptest.Server, path string, body interface{}) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", ts.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func deleteJSON(ts *httptest.Server, path string) *http.Response {
	req, _ := http.NewRequest("DELETE", ts.URL+path, nil)
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func getJSON(ts *httptest.Server, path string) *http.Response {
	resp, _ := http.Get(ts.URL + path)
	return resp
}

// --- PLN-01: Create Plan Tests ---

func TestCreatePlan_WaveMode_201(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "CREATED", result["status"])
	assert.Equal(t, "WAVE", result["mode"])
	assert.Equal(t, "Test Plan", result["name"])

	// Verify event emitted
	evts := pub.Events()
	require.Len(t, evts, 1)
	assert.Equal(t, "plan.created", evts[0].EventType())
}

func TestCreatePlan_DynamicMode_201(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	resp := postJSON(ts, "/v1/plans", createPlanPayload("DYNAMIC"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "CREATED", result["status"])
	assert.Equal(t, "DYNAMIC", result["mode"])
}

func TestCreatePlan_EmptyName_422(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	payload := createPlanPayload("WAVE")
	payload["name"] = ""

	resp := postJSON(ts, "/v1/plans", payload)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestCreatePlan_InvalidMode_422(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	payload := createPlanPayload("INVALID_MODE")

	resp := postJSON(ts, "/v1/plans", payload)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// --- PLN-02: Add Items to Plan Tests ---

func TestAddItem_WavePlan_201(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create WAVE plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	pub.Reset()

	// Add item
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	var item map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&item)
	assert.NotEmpty(t, item["id"])
	assert.Equal(t, "order-123", item["orderId"])
	assert.Equal(t, "SKU-001", item["sku"])
	assert.Equal(t, float64(5), item["quantity"])

	// Verify event emitted
	evts := pub.Events()
	require.Len(t, evts, 1)
	assert.Equal(t, "plan.item_added", evts[0].EventType())
}

func TestAddItem_DynamicPlanAutoReleases_201(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create DYNAMIC plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("DYNAMIC"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	pub.Reset()

	// Add first item - should auto-release
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-456", "SKU-002", 3))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	// Verify plan is now RELEASED
	resp3 := getJSON(ts, "/v1/plans/"+planID)
	defer resp3.Body.Close()
	var p map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&p)
	assert.Equal(t, "RELEASED", p["status"])

	// Verify events: item_added, released, status_changed
	evts := pub.Events()
	require.Len(t, evts, 3)
	assert.Equal(t, "plan.item_added", evts[0].EventType())
	assert.Equal(t, "plan.released", evts[1].EventType())
	assert.Equal(t, "plan.status_changed", evts[2].EventType())
}

func TestAddItem_PlanFull_422(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plan with maxItems=1
	payload := createPlanPayload("WAVE")
	payload["maxItems"] = 1
	resp := postJSON(ts, "/v1/plans", payload)
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	// Add first item
	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-1", "SKU-1", 1)).Body.Close()

	// Try to add second item
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-2", "SKU-2", 1))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp2.StatusCode)
}

func TestAddItem_DuplicateOrderSku_409(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plan and add item
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()

	// Try to add same order+sku
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 10))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestAddItem_CompletedPlan_409(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create, add item, process, release, complete
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-1", "SKU-1", 1)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/process", nil).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/release", nil).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/complete", nil).Body.Close()

	// Try to add item to completed plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-2", "SKU-2", 1))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

// --- PLN-03: Remove Item Tests ---

func TestRemoveItem_ValidItem_204(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create plan and add item
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	resp2 := postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5))
	defer resp2.Body.Close()
	var item map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&item)
	itemID := item["id"].(string)

	pub.Reset()

	// Remove item
	resp3 := deleteJSON(ts, "/v1/plans/"+planID+"/items/"+itemID)
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp3.StatusCode)

	// Verify event emitted
	evts := pub.Events()
	require.Len(t, evts, 1)
	assert.Equal(t, "plan.item_removed", evts[0].EventType())
}

func TestRemoveItem_NonExistentItem_404(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	// Try to remove non-existent item
	resp2 := deleteJSON(ts, "/v1/plans/"+planID+"/items/fake-item-id")
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// --- PLN-04: Process Plan Tests ---

func TestProcessPlan_WavePlanWithItems_200(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create plan and add item
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()

	pub.Reset()

	// Process plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/process", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, "PROCESSING", p["status"])

	// Verify events
	evts := pub.Events()
	require.Len(t, evts, 2)
	assert.Equal(t, "plan.processed", evts[0].EventType())
	assert.Equal(t, "plan.status_changed", evts[1].EventType())
}

func TestProcessPlan_EmptyPlan_422(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plan without items
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	// Try to process empty plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/process", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp2.StatusCode)
}

func TestProcessPlan_ReleasedPlan_409(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create DYNAMIC plan and add item (auto-releases)
	resp := postJSON(ts, "/v1/plans", createPlanPayload("DYNAMIC"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()

	// Try to process released plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/process", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

// --- PLN-05: Hold and Resume Tests ---

func TestHoldPlan_ProcessingPlan_200(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create, add item, process
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/process", nil).Body.Close()

	pub.Reset()

	// Hold plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/hold", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, "HELD", p["status"])

	// Verify events
	evts := pub.Events()
	require.Len(t, evts, 2)
	assert.Equal(t, "plan.held", evts[0].EventType())
}

func TestResumePlan_HeldPlan_200(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create, add item, process, hold
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/process", nil).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/hold", nil).Body.Close()

	// Resume plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/resume", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, "PROCESSING", p["status"])
}

// --- PLN-06: Release Plan Tests ---

func TestReleasePlan_ProcessingPlan_200(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create, add item, process
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/process", nil).Body.Close()

	pub.Reset()

	// Release plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/release", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, "RELEASED", p["status"])

	// Verify plan.released event emitted
	evts := pub.Events()
	require.Len(t, evts, 2)
	assert.Equal(t, "plan.released", evts[0].EventType())
}

func TestReleasePlan_WaveFromCreatedFails_409(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create WAVE plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	// Try to release directly from CREATED (must process first for WAVE)
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/release", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

// --- PLN-07: Complete Plan Tests ---

func TestCompletePlan_ReleasedPlan_200(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create, add item, process, release
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/process", nil).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/release", nil).Body.Close()

	pub.Reset()

	// Complete plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/complete", nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, "COMPLETED", p["status"])

	// Verify events
	evts := pub.Events()
	require.Len(t, evts, 2)
	assert.Equal(t, "plan.completed", evts[0].EventType())
}

// --- PLN-08: Cancel Plan Tests ---

func TestCancelPlan_ValidTransition_200(t *testing.T) {
	ts, _, pub := setupTestServer(t)

	// Create plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	pub.Reset()

	// Cancel plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/cancel", cancelPlanPayload("No longer needed"))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, "CANCELLED", p["status"])
	assert.Equal(t, "No longer needed", p["cancellationReason"])

	// Verify plan.cancelled event emitted
	evts := pub.Events()
	require.Len(t, evts, 2)
	assert.Equal(t, "plan.cancelled", evts[0].EventType())
}

func TestCancelPlan_CompletedPlan_409(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create, add item, process, release, complete
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-123", "SKU-001", 5)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/process", nil).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/release", nil).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/complete", nil).Body.Close()

	// Try to cancel completed plan
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/cancel", cancelPlanPayload("Test reason"))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestCancelPlan_EmptyReason_422(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	// Try to cancel with empty reason
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/cancel", cancelPlanPayload(""))
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp2.StatusCode)
}

// --- PLN-09: Query Plans Tests ---

func TestGetPlan_ValidID_200(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)

	// Get plan
	resp2 := getJSON(ts, "/v1/plans/"+planID)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var p map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&p)
	assert.Equal(t, planID, p["id"])
	assert.Equal(t, "Test Plan", p["name"])
}

func TestGetPlan_NonExistentID_404(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	resp := getJSON(ts, "/v1/plans/non-existent-id")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestListPlans_FilterByStatus_200(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create multiple plans
	resp1 := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp1.Body.Close()
	var plan1 map[string]interface{}
	json.NewDecoder(resp1.Body).Decode(&plan1)

	resp2 := postJSON(ts, "/v1/plans", createPlanPayload("DYNAMIC"))
	defer resp2.Body.Close()

	// Add item to DYNAMIC plan (auto-releases)
	var plan2 map[string]interface{}
	resp2.Body = io.NopCloser(bytes.NewReader([]byte{})) // reset
	resp2 = postJSON(ts, "/v1/plans", createPlanPayload("DYNAMIC"))
	json.NewDecoder(resp2.Body).Decode(&plan2)
	planID2 := plan2["id"].(string)
	postJSON(ts, "/v1/plans/"+planID2+"/items", addItemPayload("order-1", "SKU-1", 1)).Body.Close()

	// Query RELEASED plans
	resp3 := getJSON(ts, "/v1/plans?status=RELEASED")
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&result)
	data := result["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 1)

	// Verify pagination structure
	pagination := result["pagination"].(map[string]interface{})
	assert.NotNil(t, pagination["hasMore"])
}

func TestListPlans_WithPagination_200(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create plans
	for i := 0; i < 3; i++ {
		postJSON(ts, "/v1/plans", createPlanPayload("WAVE")).Body.Close()
	}

	// Query with limit
	resp := getJSON(ts, "/v1/plans?limit=2")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data := result["data"].([]interface{})
	assert.LessOrEqual(t, len(data), 2)
}

// --- End-to-End Lifecycle Tests ---

func TestWaveLifecycle_EndToEnd(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create WAVE plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("WAVE"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)
	assert.Equal(t, "CREATED", created["status"])

	// Add items
	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-1", "SKU-1", 1)).Body.Close()
	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-2", "SKU-2", 2)).Body.Close()

	// Process
	resp2 := postJSON(ts, "/v1/plans/"+planID+"/process", nil)
	defer resp2.Body.Close()
	var processed map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&processed)
	assert.Equal(t, "PROCESSING", processed["status"])

	// Release
	resp3 := postJSON(ts, "/v1/plans/"+planID+"/release", nil)
	defer resp3.Body.Close()
	var released map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&released)
	assert.Equal(t, "RELEASED", released["status"])

	// Complete
	resp4 := postJSON(ts, "/v1/plans/"+planID+"/complete", nil)
	defer resp4.Body.Close()
	var completed map[string]interface{}
	json.NewDecoder(resp4.Body).Decode(&completed)
	assert.Equal(t, "COMPLETED", completed["status"])
}

func TestDynamicLifecycle_EndToEnd(t *testing.T) {
	ts, _, _ := setupTestServer(t)

	// Create DYNAMIC plan
	resp := postJSON(ts, "/v1/plans", createPlanPayload("DYNAMIC"))
	defer resp.Body.Close()
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	planID := created["id"].(string)
	assert.Equal(t, "CREATED", created["status"])

	// Add first item - should auto-release
	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-1", "SKU-1", 1)).Body.Close()

	// Verify released
	resp2 := getJSON(ts, "/v1/plans/" + planID)
	defer resp2.Body.Close()
	var released map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&released)
	assert.Equal(t, "RELEASED", released["status"])

	// Add more items (continuous streaming)
	postJSON(ts, "/v1/plans/"+planID+"/items", addItemPayload("order-2", "SKU-2", 2)).Body.Close()

	// Complete
	resp3 := postJSON(ts, "/v1/plans/"+planID+"/complete", nil)
	defer resp3.Body.Close()
	var completed map[string]interface{}
	json.NewDecoder(resp3.Body).Decode(&completed)
	assert.Equal(t, "COMPLETED", completed["status"])
}
