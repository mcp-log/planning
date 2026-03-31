---
layout: default
title: Event Catalog
parent: Events
nav_order: 1
---

# Event Catalog
{: .no_toc }

All domain events published by the Planning service to Apache Kafka.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

The Planning service publishes **10 domain events** to Kafka following the event-carried state transfer pattern. Each event contains the full aggregate state for eventual consistency.

**Event Publishing Strategy**:
- Events collected in Plan aggregate during mutations
- Published transactionally after persistence succeeds
- **Kafka message key**: Plan UUID (for partition affinity and ordering)
- **Kafka producer acks**: `RequireAll` (durability)

---

## Event Summary

| Event Type | Kafka Topic | Trigger | Payload Size |
|-----------|-------------|---------|--------------|
| `plan.created` | `oms.planning.created` | `NewPlan()` | ~500 bytes |
| `plan.item_added` | `oms.planning.item-added` | `AddItem()` | ~400 bytes |
| `plan.item_removed` | `oms.planning.item-removed` | `RemoveItem()` | ~350 bytes |
| `plan.processed` | `oms.planning.processed` | `Process()` | ~350 bytes |
| `plan.held` | `oms.planning.held` | `Hold()` | ~300 bytes |
| `plan.resumed` | `oms.planning.resumed` | `Resume()` | ~300 bytes |
| `plan.released` | `oms.planning.released` | `Release()` or auto | ~400 bytes |
| `plan.completed` | `oms.planning.completed` | `Complete()` | ~350 bytes |
| `plan.cancelled` | `oms.planning.cancelled` | `Cancel(reason)` | ~450 bytes |
| `plan.status_changed` | `oms.planning.status-changed` | All transitions | ~350 bytes |

---

## Common Event Structure

All events follow this base structure:

```json
{
  "eventId": "01HZQYC1D2E3F4G5H6I7J8K9L0",
  "eventType": "plan.{action}",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:00:00Z",
  "version": 1,
  "data": { /* event-specific payload */ }
}
```

**Fields**:
- `eventId` (UUID v7): Unique event identifier (time-sortable)
- `eventType` (string): Event type discriminator
- `aggregateId` (UUID): Plan ID
- `aggregateType` (string): Always `"plan"`
- `occurredAt` (RFC3339): Event timestamp (UTC)
- `version` (int): Event schema version (currently 1)
- `data` (object): Event-specific data

---

## 1. plan.created

**Trigger**: New plan created via `NewPlan()`

**Topic**: `oms.planning.created`

**Payload**:
```json
{
  "eventId": "01HZQYC1D2E3F4G5H6I7J8K9L0",
  "eventType": "plan.created",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:00:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "name": "Morning Wave - FedEx",
    "mode": "WAVE",
    "groupingStrategy": "CARRIER",
    "priority": "HIGH",
    "status": "CREATED",
    "maxItems": 100,
    "notes": "FedEx pickup at 2pm",
    "createdAt": "2026-03-29T10:00:00Z"
  }
}
```

---

## 2. plan.item_added

**Trigger**: Item added to plan via `AddItem()`

**Topic**: `oms.planning.item-added`

**Payload**:
```json
{
  "eventId": "01HZQYC2E3F4G5H6I7J8K9L0M1",
  "eventType": "plan.item_added",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:05:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "itemId": "01HZQYA1B2C3D4E5F6G7H8I9J0",
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "quantity": 5,
    "addedAt": "2026-03-29T10:05:00Z",
    "currentItemCount": 1
  }
}
```

**Note**: For DYNAMIC plans, this event is immediately followed by `plan.released` (auto-release).

---

## 3. plan.released

**Trigger**: Plan released via `Release()` or DYNAMIC auto-release

**Topic**: `oms.planning.released`

**Payload**:
```json
{
  "eventId": "01HZQYC3F4G5H6I7J8K9L0M1N2",
  "eventType": "plan.released",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:15:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "releasedAt": "2026-03-29T10:15:00Z",
    "itemCount": 5,
    "mode": "WAVE"
  }
}
```

**Use Case**: Downstream WMS (Warehouse Management System) subscribes to this event to start pick/pack operations.

---

## 4. plan.completed

**Trigger**: Plan completed via `Complete()`

**Topic**: `oms.planning.completed`

**Payload**:
```json
{
  "eventId": "01HZQYC4G5H6I7J8K9L0M1N2O3",
  "eventType": "plan.completed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T11:00:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "completedAt": "2026-03-29T11:00:00Z",
    "itemCount": 5
  }
}
```

---

## 5. plan.cancelled

**Trigger**: Plan cancelled via `Cancel(reason)`

**Topic**: `oms.planning.cancelled`

**Payload**:
```json
{
  "eventId": "01HZQYC5H6I7J8K9L0M1N2O3P4",
  "eventType": "plan.cancelled",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:30:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "cancelledAt": "2026-03-29T10:30:00Z",
    "reason": "Carrier pickup cancelled",
    "previousStatus": "PROCESSING"
  }
}
```

---

## 6. plan.status_changed

**Trigger**: All status transitions

**Topic**: `oms.planning.status-changed`

**Payload**:
```json
{
  "eventId": "01HZQYC6I7J8K9L0M1N2O3P4Q5",
  "eventType": "plan.status_changed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:10:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "oldStatus": "CREATED",
    "newStatus": "PROCESSING",
    "changedAt": "2026-03-29T10:10:00Z"
  }
}
```

**Note**: This is a **meta-event** emitted for every status transition. Useful for audit trails and state change dashboards.

---

## Consuming Events

### Example: Go Consumer with segmentio/kafka-go

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers:  []string{"localhost:9093"},
    Topic:    "oms.planning.released",
    GroupID:  "warehouse-service",
    MinBytes: 10e3, // 10KB
    MaxBytes: 10e6, // 10MB
})

for {
    msg, err := reader.ReadMessage(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Message key is plan UUID
    planID := string(msg.Key)

    // Parse event
    var event PlanReleasedEvent
    json.Unmarshal(msg.Value, &event)

    // Process event
    handlePlanReleased(event)
}
```

### Example: Python Consumer with kafka-python

```python
from kafka import KafkaConsumer
import json

consumer = KafkaConsumer(
    'oms.planning.released',
    bootstrap_servers=['localhost:9093'],
    group_id='warehouse-service',
    value_deserializer=lambda m: json.loads(m.decode('utf-8'))
)

for message in consumer:
    plan_id = message.key.decode('utf-8')
    event = message.value
    handle_plan_released(event)
```

---

## Event Ordering Guarantees

**Partition Key**: All events for a single plan use the **plan UUID** as the Kafka message key.

**Guarantees**:
- ✅ **Per-Plan Ordering**: Events for Plan A arrive in order (guaranteed by single partition)
- ✅ **Idempotency**: Event IDs (UUID v7) can be used for deduplication
- ❌ **Cross-Plan Ordering**: Events for Plan A and Plan B may interleave

---

## Schema Evolution

**Current Version**: `v1`

Future schema changes will:
1. Add new optional fields (backward compatible)
2. Deprecate fields with 6-month sunset period
3. Increment `version` field for breaking changes
4. Maintain separate Kafka topics for v1 vs v2

---

## Full Event Specifications

For complete event schemas with all fields, see:
- [Event Contracts](https://github.com/mcp-log/planning/blob/main/specs/002-planning/contracts/events.md)
- [Kafka Topics Guide](/planning/events/kafka-topics)
