package presence

import (
	"testing"
	"time"

	"qim/internal/actor"
)

func TestPresenceActorGatewayLifecycle_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	presenceRef, err := engine.Spawn("presence-test", NewPresenceActor(engine, nil))
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	gatewayRef, err := engine.Spawn("gateway-test", actorFunc(func(ctx actor.Context) {}))
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}

	if err := presenceRef.Tell(UserConnected{UID: 1001, Gateway: gatewayRef}); err != nil {
		t.Fatalf("tell connected: %v", err)
	}
	assertGateways(t, presenceRef, 1001, 1)

	if err := presenceRef.Tell(UserDisconnected{UID: 1001, Gateway: gatewayRef}); err != nil {
		t.Fatalf("tell disconnected: %v", err)
	}
	assertGateways(t, presenceRef, 1001, 0)
}

func TestPresenceActorIgnoresInvalidMessages_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	presenceRef, err := engine.Spawn("presence-invalid-test", NewPresenceActor(engine, nil))
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	if err := presenceRef.Tell(UserConnected{}); err != nil {
		t.Fatalf("tell invalid connected: %v", err)
	}
	if err := presenceRef.Tell(UserDisconnected{}); err != nil {
		t.Fatalf("tell invalid disconnected: %v", err)
	}
	assertGateways(t, presenceRef, 0, 0)
}

func TestPresenceActorTerminatedRemovesGateway_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	presenceRef, err := engine.Spawn("presence-terminated-test", NewPresenceActor(engine, nil))
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	gatewayRef, err := engine.Spawn("gateway-terminated-test", actorFunc(func(ctx actor.Context) {}))
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	if err := presenceRef.Tell(UserConnected{UID: 1001, Gateway: gatewayRef}); err != nil {
		t.Fatalf("tell connected: %v", err)
	}
	assertGateways(t, presenceRef, 1001, 1)
	if err := presenceRef.Tell(actor.Terminated{Who: gatewayRef}); err != nil {
		t.Fatalf("tell terminated: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	assertGateways(t, presenceRef, 1001, 0)
	if err := presenceRef.Tell(actor.Terminated{}); err != nil {
		t.Fatalf("tell nil terminated: %v", err)
	}
}

func assertGateways(t *testing.T, ref *actor.ActorRef, uid uint64, want int) {
	t.Helper()
	raw, err := ref.Ask(GetGatewaysQuery{UID: uid}, time.Second)
	if err != nil {
		t.Fatalf("ask gateways: %v", err)
	}
	result := raw.(GatewaysResult)
	if len(result.Gateways) != want {
		t.Fatalf("gateways len = %d, want %d", len(result.Gateways), want)
	}
}

type actorFunc func(ctx actor.Context)

func (f actorFunc) Receive(ctx actor.Context) { f(ctx) }
