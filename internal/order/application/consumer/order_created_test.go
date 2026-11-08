package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"eventflow/internal/order/domain/entity"
	"eventflow/pkg/pubsub"
)

type fakeStore struct {
	calls   int
	rejects int
	err     error
}

func (s *fakeStore) Process(context.Context, entity.OrderCreatedEvent, int) error {
	s.calls++
	return s.err
}
func (s *fakeStore) Fail(context.Context, entity.OrderCreatedEvent, int) error { return s.err }
func (s *fakeStore) Reject(context.Context, string) error                      { s.rejects++; return s.err }

func TestHandleValidEvent(t *testing.T) {
	event := entity.OrderCreatedEvent{EventID: "b5b7d67a-bfc5-483a-a80d-7de80cab748e", OrderID: "ee779246-0d09-444b-a678-c52f72e01e5c", EventType: entity.OrderCreatedEventType, OccurredAt: time.Now()}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{}
	disposition, err := New(store).Handle(context.Background(), pubsub.Delivery{EventID: event.EventID, Type: event.EventType, Payload: payload})
	if err != nil || disposition != pubsub.Ack || store.calls != 1 {
		t.Fatalf("disposition=%v calls=%d err=%v", disposition, store.calls, err)
	}
}

func TestHandleInvalidEventRecordsRejection(t *testing.T) {
	store := &fakeStore{}
	disposition, err := New(store).Handle(context.Background(), pubsub.Delivery{Payload: []byte("invalid")})
	if err != nil || disposition != pubsub.NackDrop || store.rejects != 1 || store.calls != 0 {
		t.Fatalf("disposition=%v store=%+v err=%v", disposition, store, err)
	}
}

func TestHandleInvalidEventDoesNotAckWhenRejectionFails(t *testing.T) {
	want := errors.New("database unavailable")
	disposition, err := New(&fakeStore{err: want}).Handle(context.Background(), pubsub.Delivery{})
	if disposition != pubsub.KeepUnconfirmed || !errors.Is(err, want) {
		t.Fatalf("disposition=%v err=%v", disposition, err)
	}
}
