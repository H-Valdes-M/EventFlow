package app

import (
	"context"
	"errors"
	"testing"

	"eventflow/pkg/pubsub"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogDeliverySuccess(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	handler := logDelivery(zap.New(core), func(context.Context, pubsub.Delivery) (pubsub.Disposition, error) {
		return pubsub.Ack, nil
	})
	result, err := handler(context.Background(), pubsub.Delivery{
		MessageID: "message-1", Payload: []byte(`{"event_id":"event-1","order_id":"order-1","event_type":"order.created"}`),
	})
	if err != nil || result != pubsub.Ack || logs.Len() != 1 {
		t.Fatalf("disposition=%v err=%v logs=%d", result, err, logs.Len())
	}
	fields := logs.All()[0].ContextMap()
	if fields["event_id"] != "event-1" || fields["order_id"] != "order-1" || fields["message_id"] != "message-1" {
		t.Fatalf("campos inesperados: %v", fields)
	}
}

func TestLogDeliveryDoesNotExposeErrorOrPayload(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	want := errors.New("secret database password")
	handler := logDelivery(zap.New(core), func(context.Context, pubsub.Delivery) (pubsub.Disposition, error) {
		return pubsub.KeepUnconfirmed, want
	})
	result, err := handler(context.Background(), pubsub.Delivery{Payload: []byte("secret payload")})
	if result != pubsub.KeepUnconfirmed || !errors.Is(err, want) || logs.Len() != 1 {
		t.Fatalf("disposition=%v err=%v logs=%d", result, err, logs.Len())
	}
	if len(logs.All()[0].Context) != 0 {
		t.Fatal("el log de error no debe incluir datos sensibles")
	}
}
