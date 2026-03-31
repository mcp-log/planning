package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-log/planning/internal/domain/plan"
	"github.com/mcp-log/planning/pkg/events"
)

// mockKafkaWriter is a test double for kafka.Writer that records written messages.
type mockKafkaWriter struct {
	messages []kafka.Message
	writeErr error
}

func (m *mockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.messages = append(m.messages, msgs...)
	return nil
}

func (m *mockKafkaWriter) Close() error {
	return nil
}

func TestKafkaEventPublisher_Publish_SerializesToJSON(t *testing.T) {
	mock := &mockKafkaWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt := plan.PlanCreated{
		BaseEvent:        events.NewBaseEvent("plan.created", "plan-123", "Plan"),
		PlanID:           "plan-123",
		Name:             "Test Plan",
		Mode:             plan.Wave,
		GroupingStrategy: plan.StrategyCarrier,
		Priority:         plan.PriorityNormal,
		MaxItems:         100,
	}

	err := pub.Publish(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, mock.messages, 1)

	msg := mock.messages[0]
	var payload map[string]interface{}
	err = json.Unmarshal(msg.Value, &payload)
	require.NoError(t, err)

	assert.Equal(t, "plan.created", payload["Type"])
	assert.Equal(t, "plan-123", payload["AggregateId"])
}

func TestKafkaEventPublisher_Publish_SetsMessageKey(t *testing.T) {
	mock := &mockKafkaWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt := plan.PlanCancelled{
		BaseEvent:      events.NewBaseEvent("plan.cancelled", "plan-456", "Plan"),
		PlanID:         "plan-456",
		Reason:         "No longer needed",
		PreviousStatus: plan.Created,
		CancelledAt:    time.Now().UTC(),
	}

	err := pub.Publish(context.Background(), evt)
	require.NoError(t, err)
	require.Len(t, mock.messages, 1)

	msg := mock.messages[0]
	assert.Equal(t, "plan-456", string(msg.Key))
}

func TestKafkaEventPublisher_Publish_DerivesTopicFromEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantTopic string
	}{
		{
			name:      "plan created",
			eventType: "plan.created",
			wantTopic: "oms.planning.created",
		},
		{
			name:      "plan item added",
			eventType: "plan.item_added",
			wantTopic: "oms.planning.item-added",
		},
		{
			name:      "plan item removed",
			eventType: "plan.item_removed",
			wantTopic: "oms.planning.item-removed",
		},
		{
			name:      "plan processed",
			eventType: "plan.processed",
			wantTopic: "oms.planning.processed",
		},
		{
			name:      "plan held",
			eventType: "plan.held",
			wantTopic: "oms.planning.held",
		},
		{
			name:      "plan resumed",
			eventType: "plan.resumed",
			wantTopic: "oms.planning.resumed",
		},
		{
			name:      "plan released",
			eventType: "plan.released",
			wantTopic: "oms.planning.released",
		},
		{
			name:      "plan completed",
			eventType: "plan.completed",
			wantTopic: "oms.planning.completed",
		},
		{
			name:      "plan cancelled",
			eventType: "plan.cancelled",
			wantTopic: "oms.planning.cancelled",
		},
		{
			name:      "plan status changed",
			eventType: "plan.status_changed",
			wantTopic: "oms.planning.status-changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKafkaWriter{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			pub := &EventPublisher{
				writer: mock,
				logger: logger,
			}

			evt := events.NewBaseEvent(tt.eventType, "plan-123", "Plan")

			err := pub.Publish(context.Background(), evt)
			require.NoError(t, err)
			require.Len(t, mock.messages, 1)

			msg := mock.messages[0]
			assert.Equal(t, tt.wantTopic, msg.Topic)
		})
	}
}

func TestKafkaEventPublisher_Publish_HandlesKafkaWriteError(t *testing.T) {
	mock := &mockKafkaWriter{
		writeErr: errors.New("kafka connection failed"),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt := plan.PlanCreated{
		BaseEvent: events.NewBaseEvent("plan.created", "plan-123", "Plan"),
		PlanID:    "plan-123",
	}

	err := pub.Publish(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kafka connection failed")
}

func TestKafkaEventPublisher_PublishBatch_MultipleEvents(t *testing.T) {
	mock := &mockKafkaWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pub := &EventPublisher{
		writer: mock,
		logger: logger,
	}

	evt1 := plan.PlanReleased{
		BaseEvent:  events.NewBaseEvent("plan.released", "plan-123", "Plan"),
		PlanID:     "plan-123",
		Mode:       plan.Wave,
		ItemCount:  5,
		ReleasedAt: time.Now().UTC(),
	}
	evt2 := plan.PlanStatusChanged{
		BaseEvent: events.NewBaseEvent("plan.status_changed", "plan-123", "Plan"),
		PlanID:    "plan-123",
		OldStatus: plan.Processing,
		NewStatus: plan.Released,
		ChangedAt: time.Now().UTC(),
	}

	err := pub.PublishBatch(context.Background(), []events.DomainEvent{evt1, evt2})
	require.NoError(t, err)
	require.Len(t, mock.messages, 2)

	assert.Equal(t, "oms.planning.released", mock.messages[0].Topic)
	assert.Equal(t, "oms.planning.status-changed", mock.messages[1].Topic)
}
