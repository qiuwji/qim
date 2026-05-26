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
	_, err := engine.Spawn("agent-gw", gw)
	if err != nil {
		t.Fatal(err)
	}

	gw.handlePush(PushNotificationCmd{BotUID: 100, Type: "message_sent"})

	if len(gw.pendingEvents) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(gw.pendingEvents))
	}
}

func TestAgentGatewayActor_PushWhenReady_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	gw := NewAgentGatewayActor()
	_, err := engine.Spawn("agent-gw", gw)
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	gw.SetupSSE(&buf, nil)
	gw.handleSSEConnected(nil)
	gw.handlePush(PushNotificationCmd{BotUID: 100, Type: "message_sent"})

	output := buf.String()
	if !strings.Contains(output, `"bot_uid":100`) {
		t.Fatalf("expected bot_uid in output, got: %s", output)
	}
}

func TestAgentGatewayActor_RateLimit_BitsUT(t *testing.T) {
	gw := NewAgentGatewayActor()

	for i := 0; i < 100; i++ {
		if !gw.CheckRate() {
			t.Fatalf("CheckRate failed at iteration %d", i)
		}
	}
	if gw.CheckRate() {
		t.Fatal("CheckRate should fail after 100 calls")
	}

	gw.rateLimitSec = time.Now().Add(-2 * time.Second).Unix()
	if !gw.CheckRate() {
		t.Fatal("CheckRate should pass after window reset")
	}
}

func TestAgentGatewayActor_SSEReadyFlag_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	gw := NewAgentGatewayActor()
	_, err := engine.Spawn("agent-gw", gw)
	if err != nil {
		t.Fatal(err)
	}

	if gw.sseReady {
		t.Fatal("expected sseReady false initially")
	}

	gw.handleSSEConnected(nil)

	if !gw.sseReady {
		t.Fatal("expected sseReady true after SSEConnected")
	}

	gw.handleSSEDisconnected(nil)

	if gw.sseReady {
		t.Fatal("expected sseReady false after SSEDisconnected")
	}
}
