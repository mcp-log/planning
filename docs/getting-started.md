---
layout: default
title: Getting Started
nav_order: 2
---

# Getting Started
{: .no_toc }

This guide will help you set up and run the Planning service locally, then create your first plan.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Prerequisites

Before you begin, ensure you have:

- **Go 1.25+** installed
- **Docker & Docker Compose** for local infrastructure
- **make** for build automation
- **golang-migrate** for database migrations

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/mcp-log/planning.git
cd planning
```

### 2. Start Infrastructure

The service requires PostgreSQL (database) and Kafka (event streaming):

```bash
make docker-up
```

This starts:
- **PostgreSQL** on port `5433`
- **Kafka** on port `9093`

Wait ~10 seconds for services to be healthy.

### 3. Run Database Migrations

```bash
make migrate-up
```

This creates the `plans` and `plan_items` tables.

### 4. Start the Service

```bash
make run
```

The service will start on `http://localhost:8081`.

### 5. Verify Health

```bash
curl http://localhost:8081/health
```

Expected response: `{"status": "healthy"}`

---

## Your First WAVE Plan

Let's create a batch-based plan, add items, and release it.

### Step 1: Create a Plan

```bash
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Morning Wave - FedEx",
    "mode": "WAVE",
    "groupingStrategy": "CARRIER",
    "priority": "HIGH",
    "maxItems": 100
  }'
```

**Response**: `201 Created`
```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "name": "Morning Wave - FedEx",
  "mode": "WAVE",
  "status": "CREATED",
  "itemCount": 0
}
```

**Save the `id`** for the next steps!

### Step 2: Add Items

Add two items to the plan:

```bash
# First item
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "quantity": 5
  }'

# Second item
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8G8H9I0J1K2L3M4N5O6P",
    "sku": "GADGET-042",
    "quantity": 10
  }'
```

**Status**: Plan remains in `CREATED` status (WAVE mode doesn't auto-release).

### Step 3: Process the Plan

Validate and prepare the batch:

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/process
```

**Response**: Plan transitions to `PROCESSING` status.

### Step 4: Release to Warehouse

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/release
```

**Response**: Plan transitions to `RELEASED` status.

A `plan.released` event is published to Kafka topic `oms.planning.released`.

### Step 5: Complete the Plan

After warehouse work is done:

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/complete
```

**Response**: Plan transitions to `COMPLETED` status (terminal state).

---

## Your First DYNAMIC Plan

DYNAMIC plans auto-release when the first item is added (waveless fulfillment).

### Create DYNAMIC Plan

```bash
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Rush Order - Single Item",
    "mode": "DYNAMIC",
    "groupingStrategy": "NONE",
    "priority": "RUSH"
  }'
```

### Add First Item (Auto-Release!)

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8H9I0J1K2L3M4N5O6P7Q",
    "sku": "DOOHICKEY-999",
    "quantity": 1
  }'
```

**Result**: Plan **immediately transitions to `RELEASED`** status!

DYNAMIC plans skip `PROCESSING` and `HELD` states entirely.

---

## Listing Plans

Retrieve all plans with cursor-based pagination:

```bash
curl "http://localhost:8081/v1/plans?limit=20"
```

**Filter by status**:
```bash
curl "http://localhost:8081/v1/plans?status=RELEASED&limit=10"
```

**Filter by mode and priority**:
```bash
curl "http://localhost:8081/v1/plans?mode=WAVE&priority=HIGH"
```

---

## Advanced Scenarios

For detailed walkthroughs of:
- Capacity limits (maxItems enforcement)
- Hold and Resume flow
- Cancellation with audit trail
- Duplicate item prevention
- State transition errors

See the [Full Quick Start Guide](https://github.com/mcp-log/planning/blob/main/specs/002-planning/quickstart.md) with 7 complete scenarios.

---

## Next Steps

- [API Reference](/planning/api/v1/reference.html) - Explore all 12 endpoints
- [Architecture](/planning/architecture/) - Understand the hexagonal design
- [Event Catalog](/planning/events/catalog) - Subscribe to Kafka events
- [Domain Model](/planning/domain/data-model) - Deep dive into the Plan aggregate

---

## Development Commands

```bash
make build              # Build binary
make test               # Run all tests
make lint               # Run linters
make docker-up          # Start infrastructure
make migrate-up         # Apply migrations
make run                # Start service
```

## Environment Variables

```bash
PORT=8081
DB_HOST=localhost
DB_PORT=5433
DB_USER=planning
DB_PASSWORD=planning
DB_NAME=oms_planning
KAFKA_BROKERS=localhost:9093
```

---

## Troubleshooting

### Service won't start

**Check PostgreSQL**:
```bash
docker ps | grep planning-postgres
```

**Check migrations**:
```bash
make migrate-up
```

### Events not publishing

**Check Kafka**:
```bash
docker ps | grep planning-kafka
docker logs planning-kafka
```

### Database connection errors

Verify connection string in your environment or `.env` file matches the docker-compose settings.

---

Need help? Check the [GitHub Issues](https://github.com/mcp-log/planning/issues) or open a new one!
