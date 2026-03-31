-- Rollback Planning Service Database Schema

-- Drop trigger
DROP TRIGGER IF EXISTS trg_plans_updated_at ON plans;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_plan_items_sku;
DROP INDEX IF EXISTS idx_plan_items_order_id;
DROP INDEX IF EXISTS idx_plan_items_plan_id;
DROP INDEX IF EXISTS idx_plans_released_at;
DROP INDEX IF EXISTS idx_plans_created_at;
DROP INDEX IF EXISTS idx_plans_priority;
DROP INDEX IF EXISTS idx_plans_mode;
DROP INDEX IF EXISTS idx_plans_status;

-- Drop tables
DROP TABLE IF EXISTS plan_items CASCADE;
DROP TABLE IF EXISTS plans CASCADE;

-- Drop enums
DROP TYPE IF EXISTS plan_priority CASCADE;
DROP TYPE IF EXISTS grouping_strategy CASCADE;
DROP TYPE IF EXISTS planning_mode CASCADE;
DROP TYPE IF EXISTS plan_status CASCADE;
