package eventbus

import (
	"testing"
	"time"

	"qim/internal/actor"
)

type testEvent struct{ name string }

func (e testEvent) Name() string { return e.name }

func TestRefBusPublishSubscribe_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	busRef, err := engine.Spawn("eventbus-test", NewEventBusActor())
	if err != nil {
		t.Fatalf("spawn eventbus: %v", err)
	}
	received := make(chan Event, 1)
	subRef, err := engine.Spawn("eventbus-sub", actorFunc(func(ctx actor.Context) {
		if envelope, ok := ctx.Message().(EventEnvelope); ok {
			received <- envelope.Event
		}
	}))
	if err != nil {
		t.Fatalf("spawn subscriber: %v", err)
	}
	bus := NewRefBus(busRef)
	if err := bus.Subscribe("demo", subRef); err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}
	if err := bus.Publish(testEvent{name: "demo"}); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	select {
	case event := <-received:
		if event.Name() != "demo" {
			t.Fatalf("event = %s", event.Name())
		}
	case <-time.After(time.Second):
		t.Fatalf("subscriber did not receive event")
	}

	if err := bus.Unsubscribe("demo", subRef); err != nil {
		t.Fatalf("Unsubscribe error: %v", err)
	}
	if err := bus.Publish(testEvent{name: "demo"}); err != nil {
		t.Fatalf("Publish after unsubscribe error: %v", err)
	}
	select {
	case event := <-received:
		t.Fatalf("unexpected event after unsubscribe: %s", event.Name())
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRefBusNilInputs_BitsUT(t *testing.T) {
	var bus *RefBus
	if err := bus.Publish(testEvent{name: "demo"}); err != nil {
		t.Fatalf("nil bus Publish should be no-op: %v", err)
	}
	bus = NewRefBus(nil)
	if err := bus.Subscribe("", nil); err != nil {
		t.Fatalf("invalid Subscribe should be no-op: %v", err)
	}
	if err := bus.Unsubscribe("", nil); err != nil {
		t.Fatalf("invalid Unsubscribe should be no-op: %v", err)
	}
	if err := bus.Publish(nil); err != nil {
		t.Fatalf("nil event Publish should be no-op: %v", err)
	}
}

func TestEventBusActorTerminatedAndInvalidInputs_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	busRef, err := engine.Spawn("eventbus-terminated-test", NewEventBusActor())
	if err != nil {
		t.Fatalf("spawn eventbus: %v", err)
	}
	received := make(chan Event, 1)
	subRef, err := engine.Spawn("eventbus-terminated-sub", actorFunc(func(ctx actor.Context) {
		if envelope, ok := ctx.Message().(EventEnvelope); ok {
			received <- envelope.Event
		}
	}))
	if err != nil {
		t.Fatalf("spawn subscriber: %v", err)
	}

	if err := busRef.Tell(SubscribeCmd{}); err != nil {
		t.Fatalf("invalid subscribe tell: %v", err)
	}
	if err := busRef.Tell(UnsubscribeCmd{}); err != nil {
		t.Fatalf("invalid unsubscribe tell: %v", err)
	}
	if err := busRef.Tell(PublishCmd{}); err != nil {
		t.Fatalf("nil publish tell: %v", err)
	}
	if err := busRef.Tell(SubscribeCmd{EventName: "demo", Subscriber: subRef}); err != nil {
		t.Fatalf("subscribe tell: %v", err)
	}
	if err := busRef.Tell(actor.Terminated{Who: subRef}); err != nil {
		t.Fatalf("terminated tell: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := busRef.Tell(PublishCmd{Event: testEvent{name: "demo"}}); err != nil {
		t.Fatalf("publish tell: %v", err)
	}
	select {
	case event := <-received:
		t.Fatalf("unexpected event after terminated removal: %s", event.Name())
	case <-time.After(50 * time.Millisecond):
	}
}

type actorFunc func(ctx actor.Context)

func (f actorFunc) Receive(ctx actor.Context) { f(ctx) }
