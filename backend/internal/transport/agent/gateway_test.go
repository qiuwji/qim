package agent

import (
	"strings"
	"testing"
	"time"

	"qim/internal/actor"
)

func TestAgentGatewayActor_PushWhenNotReady_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	gw := NewAgentGatewayActor()
	ref, err := engine.Spawn("agent-gw", gw)
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	if err := ref.Tell(PushNotificationCmd{SessionID: "s1", BotUID: 100, Type: "message_sent"}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SetupSSECmd{SessionID: "s1", Writer: &buf}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SSEConnected{SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	waitGatewayIdle(t, ref)

	output := buf.String()
	if !strings.Contains(output, `"bot_uid":100`) {
		t.Fatalf("expected pending event to flush after connect, got: %s", output)
	}
}

func TestAgentGatewayActor_PushWhenReady_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	gw := NewAgentGatewayActor()
	ref, err := engine.Spawn("agent-gw", gw)
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	if err := ref.Tell(SetupSSECmd{SessionID: "s1", Writer: &buf}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SSEConnected{SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(PushNotificationCmd{SessionID: "s1", BotUID: 100, Type: "message_sent"}); err != nil {
		t.Fatal(err)
	}
	waitGatewayIdle(t, ref)

	output := buf.String()
	if !strings.Contains(output, `"bot_uid":100`) {
		t.Fatalf("expected bot_uid in output, got: %s", output)
	}
}

func TestAgentGatewayActor_RateLimit_BitsUT(t *testing.T) {
	gw := NewAgentGatewayActor()
	engine := actor.NewEngine()
	ref, err := engine.Spawn("agent-gw-rate", gw)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if !allowGateway(t, ref, "s1", 0, "") {
			t.Fatalf("CheckRate failed at iteration %d", i)
		}
	}
	if allowGateway(t, ref, "s1", 0, "") {
		t.Fatal("CheckRate should fail after 100 calls")
	}
}

func TestAgentGatewayActor_SSEReadyFlag_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	gw := NewAgentGatewayActor()
	ref, err := engine.Spawn("agent-gw-ready", gw)
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	if err := ref.Tell(SetupSSECmd{SessionID: "s1", Writer: &buf}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SSEConnected{SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SSEDisconnected{SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(PushNotificationCmd{SessionID: "s1", BotUID: 100, Type: "message_sent"}); err != nil {
		t.Fatal(err)
	}
	waitGatewayIdle(t, ref)
	if buf.String() != "" {
		t.Fatalf("expected no output while disconnected, got: %s", buf.String())
	}

	if err := ref.Tell(SetupSSECmd{SessionID: "s1", Writer: &buf}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SSEConnected{SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	waitGatewayIdle(t, ref)
	if !strings.Contains(buf.String(), `"bot_uid":100`) {
		t.Fatalf("expected queued event after reconnect, got: %s", buf.String())
	}
}

func allowGateway(t *testing.T, ref *actor.ActorRef, sessionID string, botUID uint64, toolName string) bool {
	t.Helper()
	raw, err := ref.Ask(RateLimitQuery{SessionID: sessionID, BotUID: botUID, ToolName: toolName}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	allowed, ok := raw.(bool)
	if !ok {
		t.Fatalf("expected bool rate-limit response, got %T", raw)
	}
	return allowed
}

func waitGatewayIdle(t *testing.T, ref *actor.ActorRef) {
	t.Helper()
	_ = allowGateway(t, ref, "__test_barrier__", 0, "")
}
