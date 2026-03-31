// Package postgres implements the plan.Repository port using pgx and PostgreSQL.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mcp-log/planning/internal/domain/plan"
	"github.com/mcp-log/planning/pkg/pagination"
)

// PlanRepository implements plan.Repository backed by PostgreSQL
type PlanRepository struct {
	pool *pgxpool.Pool
}

// NewPlanRepository creates a new PlanRepository backed by the given connection pool
func NewPlanRepository(pool *pgxpool.Pool) *PlanRepository {
	return &PlanRepository{pool: pool}
}

// Save persists a new plan aggregate within a transaction. It inserts the plan
// row followed by all plan item rows.
func (r *PlanRepository) Save(ctx context.Context, p *plan.Plan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback on committed tx is a no-op

	if err := r.insertPlan(ctx, tx, p); err != nil {
		return err
	}

	for _, item := range p.Items {
		if err := r.insertPlanItem(ctx, tx, item); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit tx: %w", err)
	}
	return nil
}

// FindByID retrieves a single plan by its ID, including all plan items.
// Returns plan.ErrPlanNotFound if no matching row exists.
func (r *PlanRepository) FindByID(ctx context.Context, id string) (*plan.Plan, error) {
	// Query plan
	p, err := r.findPlanByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Query items
	items, err := r.findPlanItems(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Items = items

	return p, nil
}

// Update persists mutations on an existing plan aggregate. It updates mutable
// fields: status, timestamps, cancellation_reason. It also handles item
// additions/removals by comparing the current items list with the database state.
func (r *PlanRepository) Update(ctx context.Context, p *plan.Plan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback on committed tx is a no-op

	// Update plan row
	const updateQuery = `
		UPDATE plans
		SET status              = $2,
		    updated_at          = $3,
		    processed_at        = $4,
		    released_at         = $5,
		    completed_at        = $6,
		    cancelled_at        = $7,
		    cancellation_reason = $8
		WHERE id = $1`

	tag, err := tx.Exec(ctx, updateQuery,
		p.ID,
		string(p.Status),
		p.UpdatedAt,
		p.ProcessedAt,
		p.ReleasedAt,
		p.CompletedAt,
		p.CancelledAt,
		nullableString(p.CancellationReason),
	)
	if err != nil {
		return fmt.Errorf("postgres: update plan %s: %w", p.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return plan.ErrPlanNotFound
	}

	// Sync plan items - get current items from DB
	currentItems, err := r.findPlanItems(ctx, p.ID)
	if err != nil {
		return err
	}

	// Build maps for comparison
	currentMap := make(map[string]bool)
	for _, item := range currentItems {
		currentMap[item.ID] = true
	}

	newMap := make(map[string]plan.PlanItem)
	for _, item := range p.Items {
		newMap[item.ID] = item
	}

	// Insert new items
	for _, item := range p.Items {
		if !currentMap[item.ID] {
			if err := r.insertPlanItem(ctx, tx, item); err != nil {
				return err
			}
		}
	}

	// Delete removed items
	for _, item := range currentItems {
		if _, exists := newMap[item.ID]; !exists {
			if err := r.deletePlanItem(ctx, tx, item.ID); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit tx: %w", err)
	}
	return nil
}

// List retrieves plans matching the given filter using cursor-based pagination.
// It fetches the requested limit rows and returns a cursor for the next page if needed.
func (r *PlanRepository) List(ctx context.Context, filter plan.ListFilter, limit int, cursor string) ([]*plan.Plan, string, error) {
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}
	if limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}

	// Build the query dynamically
	query := `
		SELECT
			id, name, mode, grouping_strategy, priority, status, max_items, notes,
			created_at, updated_at, processed_at, released_at, completed_at,
			cancelled_at, cancellation_reason
		FROM plans
		WHERE 1=1`

	args := make([]any, 0, 5)
	argIdx := 1

	if cursor != "" {
		cursorID, err := pagination.DecodeCursor(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("postgres: invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND id < $%d", argIdx)
		args = append(args, cursorID)
		argIdx++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}

	if filter.Mode != nil {
		query += fmt.Sprintf(" AND mode = $%d", argIdx)
		args = append(args, string(*filter.Mode))
		argIdx++
	}

	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, string(*filter.Priority))
		argIdx++
	}

	query += " ORDER BY id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("postgres: query plans: %w", err)
	}
	defer rows.Close()

	var plans []*plan.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, "", err
		}
		plans = append(plans, p)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("postgres: iterate plans: %w", err)
	}

	// For each plan, load items
	for _, p := range plans {
		items, err := r.findPlanItems(ctx, p.ID)
		if err != nil {
			return nil, "", err
		}
		p.Items = items
	}

	// Return cursor for next page
	nextCursor := ""
	if len(plans) == limit && len(plans) > 0 {
		nextCursor = pagination.EncodeCursor(plans[len(plans)-1].ID)
	}

	return plans, nextCursor, nil
}

// --- Private helper methods ---

func (r *PlanRepository) insertPlan(ctx context.Context, tx pgx.Tx, p *plan.Plan) error {
	const query = `
		INSERT INTO plans (
			id, name, mode, grouping_strategy, priority, status, max_items, notes,
			created_at, updated_at, processed_at, released_at, completed_at,
			cancelled_at, cancellation_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := tx.Exec(ctx, query,
		p.ID,
		p.Name,
		string(p.Mode),
		string(p.GroupingStrategy),
		string(p.Priority),
		string(p.Status),
		p.MaxItems,
		nullableString(p.Notes),
		p.CreatedAt,
		p.UpdatedAt,
		p.ProcessedAt,
		p.ReleasedAt,
		p.CompletedAt,
		p.CancelledAt,
		nullableString(p.CancellationReason),
	)
	if err != nil {
		return fmt.Errorf("postgres: insert plan %s: %w", p.ID, err)
	}
	return nil
}

func (r *PlanRepository) insertPlanItem(ctx context.Context, tx pgx.Tx, item plan.PlanItem) error {
	const query = `
		INSERT INTO plan_items (id, plan_id, order_id, sku, quantity, added_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := tx.Exec(ctx, query,
		item.ID,
		item.PlanID,
		item.OrderID,
		item.SKU,
		item.Quantity,
		item.AddedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: insert plan item %s: %w", item.ID, err)
	}
	return nil
}

func (r *PlanRepository) deletePlanItem(ctx context.Context, tx pgx.Tx, itemID string) error {
	const query = `DELETE FROM plan_items WHERE id = $1`

	_, err := tx.Exec(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("postgres: delete plan item %s: %w", itemID, err)
	}
	return nil
}

func (r *PlanRepository) findPlanByID(ctx context.Context, id string) (*plan.Plan, error) {
	const query = `
		SELECT
			id, name, mode, grouping_strategy, priority, status, max_items, notes,
			created_at, updated_at, processed_at, released_at, completed_at,
			cancelled_at, cancellation_reason
		FROM plans
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	p, err := scanPlan(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, plan.ErrPlanNotFound
		}
		return nil, fmt.Errorf("postgres: query plan %s: %w", id, err)
	}
	return p, nil
}

func (r *PlanRepository) findPlanItems(ctx context.Context, planID string) ([]plan.PlanItem, error) {
	const query = `
		SELECT id, plan_id, order_id, sku, quantity, added_at
		FROM plan_items
		WHERE plan_id = $1
		ORDER BY added_at ASC`

	rows, err := r.pool.Query(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("postgres: query plan items for plan %s: %w", planID, err)
	}
	defer rows.Close()

	var items []plan.PlanItem
	for rows.Next() {
		var item plan.PlanItem
		err := rows.Scan(
			&item.ID,
			&item.PlanID,
			&item.OrderID,
			&item.SKU,
			&item.Quantity,
			&item.AddedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan plan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate plan items: %w", err)
	}

	return items, nil
}

// scannable defines the common interface for pgx.Row and pgx.Rows
type scannable interface {
	Scan(dest ...any) error
}

func scanPlan(row scannable) (*plan.Plan, error) {
	var p plan.Plan
	var mode, strategy, priority, status string
	var notes, cancellationReason *string
	var processedAt, releasedAt, completedAt, cancelledAt *time.Time

	err := row.Scan(
		&p.ID,
		&p.Name,
		&mode,
		&strategy,
		&priority,
		&status,
		&p.MaxItems,
		&notes,
		&p.CreatedAt,
		&p.UpdatedAt,
		&processedAt,
		&releasedAt,
		&completedAt,
		&cancelledAt,
		&cancellationReason,
	)
	if err != nil {
		return nil, err
	}

	p.Mode = plan.Mode(mode)
	p.GroupingStrategy = plan.Strategy(strategy)
	p.Priority = plan.Priority(priority)
	p.Status = plan.Status(status)
	p.ProcessedAt = processedAt
	p.ReleasedAt = releasedAt
	p.CompletedAt = completedAt
	p.CancelledAt = cancelledAt

	if notes != nil {
		p.Notes = *notes
	}
	if cancellationReason != nil {
		p.CancellationReason = *cancellationReason
	}

	p.Items = []plan.PlanItem{} // Initialize empty slice

	return &p, nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
