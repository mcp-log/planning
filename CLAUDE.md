# Planning — Fulfillment Planning Service

## Project Overview
A standalone fulfillment planning service extracted from the OMS monorepo. Manages
plan lifecycle, item assignment, and warehouse release strategies using wave-based
and dynamic streaming modes.

## Architecture
- **Language**: Go 1.25+
- **Pattern**: Hexagonal Architecture + DDD Aggregates + CQRS
- **API**: OpenAPI 3.0.3 with oapi-codegen (Chi router)
- **Database**: PostgreSQL 16
- **Messaging**: Apache Kafka via segmentio/kafka-go
- **Testing**: testify + in-memory repository

## Key Principles
1. **Hexagonal Architecture**: Domain has zero infrastructure dependencies
2. **DDD**: Plan aggregate enforces invariants, domain events as first-class citizens
3. **CQRS**: Separate command and query handlers
4. **Event Sourcing Pattern**: All state changes emit domain events
5. **UUID v7**: Time-sortable identifiers for all entities
6. **RFC 7807**: Structured error responses

## Domain Model

### Plan Aggregate
- **Modes**:
  - WAVE (batch): accumulate → process → release
  - DYNAMIC (streaming): auto-release on first item
- **Status Machine**: CREATED → PROCESSING → HELD → RELEASED → COMPLETED/CANCELLED
- **Grouping Strategies**: CARRIER, ZONE, PRIORITY, CHANNEL, NONE
- **Priority Levels**: LOW, NORMAL, HIGH, RUSH

### Commands (9)
1. CreatePlan
2. AddItem
3. RemoveItem
4. ProcessPlan (WAVE mode only)
5. HoldPlan
6. ResumePlan
7. ReleasePlan
8. CompletePlan
9. CancelPlan

### Queries (2)
1. GetPlan (by ID)
2. ListPlans (with cursor pagination, filters: status, mode, priority)

## Project Structure
```
planning/
  pkg/                              # Local shared kernel (owned by this service)
    events/event.go                 # Domain event interfaces
    identity/identity.go            # UUID v7 generation
    errors/errors.go                # RFC 7807 types
    pagination/cursor.go            # Pagination helpers
  internal/
    domain/plan/                    # Aggregate root, state machine, events
    app/command/                    # 9 command handlers
    app/query/                      # 2 query handlers
    ports/                          # HTTP handlers + router + tests
    adapters/postgres/              # PostgreSQL repository
    adapters/publisher/             # Kafka event publisher + tests
    service/                        # DI wiring
  cmd/planning/main.go              # Entrypoint (port 8081)
  migrations/                       # DB migrations
  api/openapi/planning.yaml         # OpenAPI spec
  specs/002-planning/               # Feature specifications
```

## Development Commands
```bash
make build              # Build the service
make test               # Run all tests
make test-unit          # Run domain tests only
make test-integration   # Run HTTP integration tests
make lint               # Run linters
make docker-up          # Start Postgres (5433) + Kafka (9093)
make migrate-up         # Apply migrations
make run                # Start service on :8081
```

## Database
- **Host**: localhost:5433
- **Database**: oms_planning
- **User/Password**: planning/planning

## Kafka Topics
- `oms.plans.created`
- `oms.plans.item-added`
- `oms.plans.item-removed`
- `oms.plans.processed`
- `oms.plans.held`
- `oms.plans.resumed`
- `oms.plans.released`
- `oms.plans.completed`
- `oms.plans.cancelled`
- `oms.plans.status-changed`

Message keys: Plan UUID (aggregateId) for partition ordering

## Test Coverage
- 27 domain tests (invariants + state machine)
- Hexagonal architecture allows adapter swaps without test changes
- All tests use in-memory repository for speed

## Environment Variables
```
PORT=8081
DB_HOST=localhost
DB_PORT=5433
DB_USER=planning
DB_PASSWORD=planning
DB_NAME=oms_planning
KAFKA_BROKERS=localhost:9093
```

## API Endpoints
See `api/openapi/planning.yaml` for full spec.

### Plans
- `POST /api/v1/plans` - Create plan
- `GET /api/v1/plans` - List plans (with cursor pagination)
- `GET /api/v1/plans/{id}` - Get plan by ID

### Plan Actions
- `POST /api/v1/plans/{id}/items` - Add item
- `DELETE /api/v1/plans/{id}/items/{itemId}` - Remove item
- `POST /api/v1/plans/{id}/process` - Process plan (WAVE only)
- `POST /api/v1/plans/{id}/hold` - Hold plan
- `POST /api/v1/plans/{id}/resume` - Resume plan
- `POST /api/v1/plans/{id}/release` - Release plan
- `POST /api/v1/plans/{id}/complete` - Complete plan
- `POST /api/v1/plans/{id}/cancel` - Cancel plan

## Deployment
Standalone service, no dependencies on OMS monorepo. Can be deployed independently.
