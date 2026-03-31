---
layout: default
title: Kafka Topics
parent: Events
nav_order: 2
---

# Kafka Topics Guide
{: .no_toc }

Comprehensive guide to Kafka topic configuration and naming conventions for the Planning service.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Topic Naming Convention

**Pattern**: `oms.planning.{event-name}`

**Rules**:
1. Strip `plan.` prefix from event type
2. Replace underscores with hyphens
3. Prepend `oms.planning.`

**Examples**:
- `plan.created` → `oms.planning.created`
- `plan.item_added` → `oms.planning.item-added`
- `plan.status_changed` → `oms.planning.status-changed`

---

## Topic List

| Topic Name | Events/sec (est.) | Retention | Partitions |
|-----------|-------------------|-----------|------------|
| `oms.planning.created` | ~10 | 30 days | 3 |
| `oms.planning.item-added` | ~100 | 30 days | 3 |
| `oms.planning.item-removed` | ~20 | 30 days | 3 |
| `oms.planning.processed` | ~10 | 30 days | 3 |
| `oms.planning.held` | ~2 | 30 days | 3 |
| `oms.planning.resumed` | ~2 | 30 days | 3 |
| `oms.planning.released` | ~10 | 90 days | 3 |
| `oms.planning.completed` | ~10 | 90 days | 3 |
| `oms.planning.cancelled` | ~5 | 90 days | 3 |
| `oms.planning.status-changed` | ~150 | 30 days | 3 |

---

## Topic Configuration

### Production Settings

```bash
# Create topics with optimal settings
kafka-topics.sh --create \
  --topic oms.planning.released \
  --partitions 3 \
  --replication-factor 3 \
  --config retention.ms=7776000000 \  # 90 days
  --config min.insync.replicas=2 \
  --config compression.type=snappy \
  --bootstrap-server localhost:9093
```

**Key Settings**:
- **Partitions**: 3 (allows 3 parallel consumers)
- **Replication**: 3 (high availability)
- **Min ISR**: 2 (durability without sacrificing availability)
- **Retention**: 30-90 days depending on topic
- **Compression**: Snappy (balance of speed and compression ratio)

### Local Development

```yaml
# docker-compose.yml
kafka:
  environment:
    KAFKA_AUTO_CREATE_TOPICS_ENABLE: 'true'
    KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
```

Topics auto-create with default settings (1 partition, 1 replica).

---

## Message Key Strategy

**All events**: Plan UUID (aggregateId)

**Benefits**:
1. **Ordering**: Events for same plan guaranteed to be in order
2. **Partition Affinity**: Same plan always routes to same partition
3. **Compaction**: Future support for log compaction (keep latest state per plan)

**Example**:
```
Message Key: "01HZQY9KT2X3FGHJK6MNPQRSTU"
Partition:   hash(key) % 3 = Partition 1
```

---

## Consumer Groups

### Recommended Consumer Groups

| Consumer Group | Topics Subscribed | Purpose |
|---------------|-------------------|---------|
| `warehouse-service` | `oms.planning.released` | Start pick/pack operations |
| `analytics-service` | All topics | Real-time dashboards & metrics |
| `audit-service` | All topics | Compliance & audit trail |
| `notification-service` | `completed`, `cancelled` | Send completion notifications |

### Offset Management

**Auto Commit**: Disabled (manual commit for at-least-once processing)

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers:        []string{"localhost:9093"},
    Topic:          "oms.planning.released",
    GroupID:        "warehouse-service",
    CommitInterval: 0, // Manual commit
})

msg, _ := reader.ReadMessage(ctx)
processEvent(msg)
reader.CommitMessages(ctx, msg) // Commit after successful processing
```

---

## Monitoring

### Key Metrics

- **Producer**: `kafka.producer.requests.total`, `kafka.producer.errors.total`
- **Consumer Lag**: `kafka.consumer.lag` (consumers falling behind)
- **Partition Balance**: Ensure even distribution across partitions

### Kafka UI (Local Development)

```bash
docker run -p 8080:8080 \
  -e KAFKA_CLUSTERS_0_NAME=local \
  -e KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=localhost:9093 \
  provectuslabs/kafka-ui:latest
```

Visit `http://localhost:8080` to view topics, partitions, and consumer groups.

---

## Troubleshooting

### Issue: Consumer Lag Growing

**Symptom**: `kafka.consumer.lag` increasing

**Solutions**:
1. Scale consumers (add more instances to consumer group)
2. Increase partition count
3. Optimize consumer processing logic

### Issue: Events Not Published

**Check**:
```bash
# List topics
kafka-topics.sh --list --bootstrap-server localhost:9093

# Consume from topic
kafka-console-consumer.sh \
  --bootstrap-server localhost:9093 \
  --topic oms.planning.released \
  --from-beginning
```

### Issue: Duplicate Events

**Solution**: Implement idempotent consumer using `eventId` (UUID v7):

```go
func handleEvent(event DomainEvent) {
    // Check if already processed
    if db.EventExists(event.EventID) {
        log.Info("Duplicate event, skipping", "eventId", event.EventID)
        return
    }

    // Process and store eventId atomically
    tx := db.Begin()
    tx.StoreEventID(event.EventID)
    tx.ProcessEvent(event)
    tx.Commit()
}
```

---

## Next Steps

- [Event Catalog](/planning/events/catalog) - Full event schemas
- [API Reference](/planning/api/v1/reference.html) - REST endpoints that trigger events
- [Architecture](/planning/architecture/) - System design
