---
layout: default
title: Home
nav_order: 1
description: "Planning Service - Fulfillment planning for the OMS ecosystem"
permalink: /
---

# Planning Service
{: .fs-9 }

Fulfillment planning service for the OMS ecosystem. Organizes confirmed orders into executable warehouse batches using WAVE or DYNAMIC modes.
{: .fs-6 .fw-300 }

[Get Started](/planning/getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View API Reference](/planning/api/v1/reference.html){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Key Features

### 🌊 WAVE Mode (Batch Processing)
Accumulate orders into batches, validate and optimize, then release as a single wave. Inspired by SAP EWM and Microsoft Dynamics 365.

**Use Cases**: Carrier consolidation, zone-based picking, labor planning

### ⚡ DYNAMIC Mode (Continuous Streaming)
Auto-release orders immediately as they're added. Waveless fulfillment for minimal cycle time. Inspired by Manhattan Associates.

**Use Cases**: Just-in-time fulfillment, single-item orders, rush orders

### 🏗 Architecture

- **Pattern**: Hexagonal Architecture + DDD + CQRS
- **Language**: Go 1.25+
- **Database**: PostgreSQL 16
- **Messaging**: Apache Kafka
- **API**: OpenAPI 3.0.3

### 📊 Planning Capabilities

- **Capacity Management**: Configure max items per plan
- **Priority-Based**: LOW, NORMAL, HIGH, RUSH priorities
- **Grouping Strategies**: CARRIER, ZONE, PRIORITY, CHANNEL
- **State Machine**: 6 states with complete audit trail
- **Event-Driven**: 10 domain events published to Kafka

---

## Quick Example

### Create a WAVE Plan

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

### Add Items

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "quantity": 5
  }'
```

### Process and Release

```bash
# Process the batch
curl -X POST http://localhost:8081/v1/plans/{planId}/process

# Release to warehouse floor
curl -X POST http://localhost:8081/v1/plans/{planId}/release
```

---

## State Machine

```
CREATED → (WAVE) → PROCESSING → RELEASED → COMPLETED
           ↓
        HELD (pause)
           ↓
      PROCESSING (resume)

CREATED → (DYNAMIC: auto on first item) → RELEASED → COMPLETED
```

**Terminal States**: COMPLETED, CANCELLED

---

## Documentation

### Getting Started
- [Quick Start Guide](/planning/getting-started) - Setup and first plan
- [API Reference](/planning/api/v1/reference.html) - Interactive OpenAPI docs

### Architecture
- [Hexagonal Architecture](/planning/architecture/hexagonal) - Ports & adapters pattern
- [DDD Patterns](/planning/architecture/ddd) - Aggregates, value objects, domain events
- [State Machine](/planning/architecture/state-machine) - Detailed state transitions

### Domain
- [Data Model](/planning/domain/data-model) - Plan aggregate structure
- [Business Rules](/planning/domain/invariants) - Domain invariants

### Events
- [Event Catalog](/planning/events/catalog) - All domain events with schemas
- [Kafka Topics](/planning/events/kafka-topics) - Topic mapping guide

---

## Part of OMS Ecosystem

| Service | Purpose | Repository |
|---------|---------|------------|
| Order Intake | Order creation & confirmation | *coming soon* |
| **Planning** | Fulfillment planning | [mcp-log/planning](https://github.com/mcp-log/planning) |
| Fulfillment | Pick, pack, ship | *future* |
| Shipping | Carrier integration | *future* |

Each bounded context is independently deployable with its own database and event stream.
