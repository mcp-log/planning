# Planning — Fulfillment Planning Service

![Go](https://img.shields.io/badge/go-1.25+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Quick Links
- [📘 Full Documentation](https://mcp-log.github.io/planning/) *(coming soon)*
- [🔌 API Reference](https://mcp-log.github.io/planning/api/v1/reference.html) *(coming soon)*
- [🚀 Quick Start](#quick-start)
- [🎯 User Stories](specs/002-planning/spec.md)

## Overview

Standalone fulfillment planning service for the OMS ecosystem. Organizes confirmed orders into executable warehouse batches using **WAVE** (batch) or **DYNAMIC** (streaming) modes.

**Key Features**:
- 🌊 **WAVE Mode**: Batch-based planning (accumulate → process → release)
- ⚡ **DYNAMIC Mode**: Continuous streaming (auto-release on first item)
- 📦 **Capacity Management**: Configurable item limits per plan
- 🔄 **State Machine**: 6 states, 9 commands, 10 events
- 🎯 **Priority-Based**: LOW, NORMAL, HIGH, RUSH prioritization
- 📊 **Grouping Strategies**: CARRIER, ZONE, PRIORITY, CHANNEL
- 📡 **Event-Driven**: Kafka integration for downstream consumers

## Architecture

- **Pattern**: Hexagonal Architecture + DDD Aggregates + CQRS
- **Language**: Go 1.25+
- **API**: OpenAPI 3.0.3 (oapi-codegen + Chi router)
- **Database**: PostgreSQL 16
- **Messaging**: Apache Kafka (segmentio/kafka-go)
- **Testing**: testify + in-memory repository

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- make
- golang-migrate (for migrations)

### Steps

1. **Clone Repository**
   ```bash
   git clone https://github.com/mcp-log/planning.git
   cd planning
   ```

2. **Start Infrastructure**
   ```bash
   make docker-up
   # Starts PostgreSQL (port 5433) + Kafka (port 9093)
   ```

3. **Run Migrations**
   ```bash
   make migrate-up
   ```

4. **Start Service**
   ```bash
   make run
   # Service runs on http://localhost:8081
   ```

5. **Verify Health**
   ```bash
   curl http://localhost:8081/health
   ```

## API Overview

### Core Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/plans` | Create new plan |
| `GET` | `/v1/plans` | List plans with cursor pagination & filters |
| `GET` | `/v1/plans/{id}` | Get plan by ID with items |
| `POST` | `/v1/plans/{id}/items` | Add item to plan |
| `DELETE` | `/v1/plans/{id}/items/{itemId}` | Remove item from plan |
| `POST` | `/v1/plans/{id}/process` | Process plan (WAVE mode only) |
| `POST` | `/v1/plans/{id}/hold` | Pause processing |
| `POST` | `/v1/plans/{id}/resume` | Resume processing |
| `POST` | `/v1/plans/{id}/release` | Release plan to warehouse floor |
| `POST` | `/v1/plans/{id}/complete` | Mark plan as completed |
| `POST` | `/v1/plans/{id}/cancel` | Cancel plan with audit reason |

Full API specification: [api/openapi/planning.yaml](api/openapi/planning.yaml)

### Examples

**Create WAVE Plan:**
```bash
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Morning Wave - FedEx Ground",
    "mode": "WAVE",
    "groupingStrategy": "CARRIER",
    "priority": "HIGH",
    "maxItems": 100,
    "notes": "FedEx pickup at 2pm"
  }'
```

**Add Item (Auto-Release for DYNAMIC):**
```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "quantity": 5
  }'
```

**Process Plan (WAVE Mode):**
```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/process
```

**Release Plan:**
```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/release
```

## Planning Modes

### WAVE Mode (Batch Processing)

Inspired by **SAP EWM** and **Microsoft Dynamics 365** wave planning.

**Lifecycle**: `CREATED → PROCESSING → RELEASED → COMPLETED`

**Use Cases**:
- Scheduled batch releases for optimized pick paths
- Carrier consolidation (combine orders for single pickup)
- Zone-based picking (group items in same warehouse area)
- Labor planning (schedule staff around wave releases)

**State Transitions**:
```
CREATED → AddItems → PROCESSING → RELEASED → COMPLETED
              ↓
           HELD (pause for issues)
              ↓
         PROCESSING (resume)
```

### DYNAMIC Mode (Continuous Streaming)

Inspired by **Manhattan Associates** waveless fulfillment and **Körber** dynamic allocation.

**Lifecycle**: `CREATED → RELEASED (auto) → COMPLETED`

**Use Cases**:
- Just-in-time fulfillment (minimize order-to-ship cycle time)
- Single-item orders (no batching benefit)
- Real-time inventory allocation
- High-priority rush orders

**State Transitions**:
```
CREATED → AddItem (first) → RELEASED (auto) → COMPLETED
```

**Key Difference**: DYNAMIC plans skip `PROCESSING` and `HELD` states entirely. First `AddItem()` call automatically releases the plan.

## State Machine

```
                 ┌─────────┐
                 │ CREATED │ (initial state)
                 └────┬────┘
                      │
          ┌───────────┼───────────┐
          │           │           │
     (WAVE mode)  (cancel)  (DYNAMIC mode: auto on first AddItem)
          │           │           │
          ▼           ▼           ▼
     ┌──────────┐ ┌──────────┐ ┌─────────┐
     │PROCESSING│ │CANCELLED │ │RELEASED │
     └─────┬────┘ └──────────┘ └────┬────┘
           │      (terminal)         │
      ┌────┼────┐                    │
      │    │    │                    │
   (hold) (release) (cancel)      (complete)
      │    │    │                    │
      ▼    ▼    ▼                    ▼
   ┌────┐ │  ┌──────────┐      ┌───────────┐
   │HELD│─┘  │CANCELLED │      │ COMPLETED │
   └────┘    └──────────┘      └───────────┘
  (resume)   (terminal)         (terminal)
```

**Terminal States**: `COMPLETED`, `CANCELLED` (no transitions out)

## Event Catalog

All events are published to Apache Kafka with the plan UUID as the message key (for partition ordering).

| Event | Kafka Topic | Trigger |
|-------|-------------|---------|
| `plan.created` | `oms.planning.created` | Plan created |
| `plan.item_added` | `oms.planning.item-added` | Item added to plan |
| `plan.item_removed` | `oms.planning.item-removed` | Item removed from plan |
| `plan.processed` | `oms.planning.processed` | Plan processed (WAVE only) |
| `plan.held` | `oms.planning.held` | Plan paused for issue resolution |
| `plan.resumed` | `oms.planning.resumed` | Plan resumed from HELD |
| `plan.released` | `oms.planning.released` | Plan released to warehouse |
| `plan.completed` | `oms.planning.completed` | Plan completed |
| `plan.cancelled` | `oms.planning.cancelled` | Plan cancelled with reason |
| `plan.status_changed` | `oms.planning.status-changed` | Any status transition |

**Message Key**: Plan UUID (aggregateId) for partition ordering.

**Event Schema**: See [specs/002-planning/contracts/events.md](specs/002-planning/contracts/events.md)

## Development Commands

### Build & Test
```bash
make build              # Build the planning service binary
make test               # Run all tests (domain + integration)
make test-unit          # Run domain tests only
make test-integration   # Run HTTP integration tests
make lint               # Run golangci-lint
```

### Database
```bash
make migrate-up         # Apply database migrations
make migrate-down       # Rollback last migration
```

### Docker
```bash
make docker-up          # Start PostgreSQL + Kafka
make docker-down        # Stop and remove containers
```

### Running Locally
```bash
make run                # Start service on :8081
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

## Documentation

- **Full Docs**: https://mcp-log.github.io/planning/ *(coming soon)*
- **API Reference**: [api/openapi/planning.yaml](api/openapi/planning.yaml)
- **Quick Start**: [specs/002-planning/quickstart.md](specs/002-planning/quickstart.md) (7 scenarios)
- **Data Model**: [specs/002-planning/data-model.md](specs/002-planning/data-model.md)
- **REST Contracts**: [specs/002-planning/contracts/rest-endpoints.md](specs/002-planning/contracts/rest-endpoints.md)
- **Event Contracts**: [specs/002-planning/contracts/events.md](specs/002-planning/contracts/events.md)

## Testing

The Planning service has comprehensive test coverage:

- **27 domain tests**: Aggregate invariants, state machine transitions
- **HTTP integration tests**: All 12 endpoints with edge cases
- **5 Kafka publisher tests**: Serialization, topic routing, error handling
- **In-memory repository**: Fast, no database dependencies for tests

Run tests:
```bash
make test               # All tests
make test-unit          # Domain tests only (~0.5s)
make test-integration   # HTTP tests with in-memory repo (~1.2s)
```

## Project Structure

```
planning/
├── pkg/                    # Local shared kernel
│   ├── events/             # Domain event interfaces
│   ├── identity/           # UUID v7 generation
│   ├── errors/             # RFC 7807 Problem Details
│   └── pagination/         # Cursor-based pagination
├── internal/
│   ├── domain/plan/        # Plan aggregate + state machine
│   ├── app/
│   │   ├── command/        # 9 command handlers (CQRS)
│   │   └── query/          # 2 query handlers (CQRS)
│   ├── ports/              # HTTP handlers (hexagonal)
│   ├── adapters/
│   │   ├── postgres/       # PostgreSQL repository
│   │   └── publisher/      # Kafka event publisher
│   └── service/            # Dependency injection wiring
├── cmd/planning/main.go    # Application entrypoint
├── api/openapi/            # OpenAPI 3.0.3 specification
├── specs/002-planning/     # Feature specs + contracts
└── migrations/             # Database migrations
```

## Part of OMS Ecosystem

This service is part of the broader OMS (Order Management System) ecosystem:

- **Order Intake**: https://github.com/mcp-log/oms *(coming soon)*
- **Planning**: https://github.com/mcp-log/planning *(this service)*
- **Fulfillment**: *(future)*
- **Shipping**: *(future)*

Each bounded context is independently deployable with its own database and event stream.

## Contributing

Contributions are welcome! Please ensure:

- All tests pass: `make test`
- Code is formatted: `make lint`
- Domain invariants are documented
- OpenAPI spec is updated for API changes

## License

MIT License - See LICENSE file for details.
