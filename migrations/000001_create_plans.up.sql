-- Planning Service Database Schema

-- Enums
CREATE TYPE plan_status AS ENUM (
    'CREATED',
    'PROCESSING',
    'HELD',
    'RELEASED',
    'COMPLETED',
    'CANCELLED'
);

CREATE TYPE planning_mode AS ENUM ('WAVE', 'DYNAMIC');

CREATE TYPE grouping_strategy AS ENUM (
    'CARRIER',
    'ZONE',
    'PRIORITY',
    'CHANNEL',
    'NONE'
);

CREATE TYPE plan_priority AS ENUM (
    'LOW',
    'NORMAL',
    'HIGH',
    'RUSH'
);

-- Plans table
CREATE TABLE plans (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL CHECK (TRIM(name) <> ''),
    mode planning_mode NOT NULL,
    grouping_strategy grouping_strategy NOT NULL DEFAULT 'NONE',
    priority plan_priority NOT NULL DEFAULT 'NORMAL',
    status plan_status NOT NULL DEFAULT 'CREATED',
    max_items INTEGER NOT NULL DEFAULT 0 CHECK (max_items >= 0),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,

    -- Constraints
    CONSTRAINT chk_cancellation_reason
        CHECK (
            (status = 'CANCELLED' AND TRIM(cancellation_reason) <> '')
            OR status <> 'CANCELLED'
        ),
    CONSTRAINT chk_processed_at
        CHECK (
            (status IN ('PROCESSING', 'HELD', 'RELEASED', 'COMPLETED') AND processed_at IS NOT NULL)
            OR status NOT IN ('PROCESSING', 'HELD')
        ),
    CONSTRAINT chk_released_at
        CHECK (
            (status IN ('RELEASED', 'COMPLETED') AND released_at IS NOT NULL)
            OR status NOT IN ('RELEASED', 'COMPLETED')
        )
);

-- Plan items table
CREATE TABLE plan_items (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    order_id UUID NOT NULL,
    sku VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: no duplicate (plan_id, order_id, sku)
    CONSTRAINT uq_plan_order_sku UNIQUE (plan_id, order_id, sku)
);

-- Indexes for plans
CREATE INDEX idx_plans_status ON plans(status);
CREATE INDEX idx_plans_mode ON plans(mode);
CREATE INDEX idx_plans_priority ON plans(priority);
CREATE INDEX idx_plans_created_at ON plans(created_at);
CREATE INDEX idx_plans_released_at ON plans(released_at) WHERE released_at IS NOT NULL;

-- Indexes for plan_items
CREATE INDEX idx_plan_items_plan_id ON plan_items(plan_id);
CREATE INDEX idx_plan_items_order_id ON plan_items(order_id);
CREATE INDEX idx_plan_items_sku ON plan_items(sku);

-- Trigger: Update updated_at on plans
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_plans_updated_at
BEFORE UPDATE ON plans
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
