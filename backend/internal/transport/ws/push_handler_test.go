package ws

import (
	"testing"
	"time"

	"qim/internal/actor"
	calldomain "qim/internal/domain/call"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	presencedomain "qim/internal/domain/presence"
	"qim/internal/eventbus"
)

func TestRoutePushEvent_BitsUT(t *testing.T) {
	tests := []struct {
		name       string
		event      interface{ Name() string }
		wantType   string
		wantAction string
		wantUIDs   int
	}{
		{"新消息", conversation.MessageSentEvent{MemberUIDs: []uint64{1, 2}}, "message", "new", 2},
		{"撤回消息", conversation.MessageRevokedEvent{MemberUIDs: []uint64{1}}, "message", "revoked", 1},
		{"会话更新", conversation.ConversationUpdatedEvent{MemberUIDs: []uint64{1, 2}}, "conversation", "updated", 2},
		{"成员入群", conversation.MemberJoinedEvent{MemberUIDs: []uint64{1, 2, 3}}, "member", "joined", 3},
		{"成员退群", conversation.MemberLeftEvent{MemberUIDs: []uint64{1}}, "member", "left", 1},
		{"成员被踢", conversation.MemberKickedEvent{MemberUIDs: []uint64{1}}, "member", "kicked", 1},
		{"群主转让", conversation.OwnerTransferredEvent{MemberUIDs: []uint64{1, 2}}, "member", "owner_transferred", 2},
		{"群解散", conversation.GroupDissolvedEvent{MemberUIDs: []uint64{1, 2}}, "conversation", "group_dissolved", 2},
		{"好友申请", friend.FriendRequestCreatedEvent{ToUID: 2}, "friend", "request", 1},
		{"好友同意", friend.FriendRequestHandledEvent{FromUID: 1, Accepted: true}, "friend", "accepted", 1},
		{"好友拒绝", friend.FriendRequestHandledEvent{FromUID: 1, Accepted: false}, "friend", "rejected", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pushType, action, recipients, data, ok := routePushEvent(tt.event)
			if !ok {
				t.Fatalf("routePushEvent should match")
			}
			if pushType != tt.wantType || action != tt.wantAction || len(recipients) != tt.wantUIDs || data == nil {
				t.Fatalf("got type=%s action=%s recipients=%v data=%T", pushType, action, recipients, data)
			}
		})
	}
}

func TestUniqueUIDs_BitsUT(t *testing.T) {
	got := uniqueUIDs([]uint64{1, 2, 1, 0, 2})
	if len(got) != 2 {
		t.Fatalf("unique uids = %+v", got)
	}
}

func TestMessagePushHandlerReceive_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	received := make(chan PushCmd, 2)
	gateway, err := engine.Spawn("push-gateway-test", pushGatewayActor{received: received})
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	presence, err := engine.Spawn("push-presence-test", pushPresenceActor{gateway: gateway})
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	handler, err := engine.Spawn("push-handler-test", NewMessagePushActor(presence, nil))
	if err != nil {
		t.Fatalf("spawn handler: %v", err)
	}

	event := conversation.MessageSentEvent{ConversationID: 1, MemberUIDs: []uint64{1, 1, 0}, Content: "hello"}
	if err := handler.Tell(eventbus.EventEnvelope{Event: event}); err != nil {
		t.Fatalf("tell event: %v", err)
	}
	select {
	case cmd := <-received:
		if cmd.Type != "message" || cmd.Action != "new" {
			t.Fatalf("push cmd = %+v", cmd)
		}
	case <-time.After(time.Second):
		t.Fatalf("gateway did not receive push")
	}

	if err := handler.Tell(conversation.TypingPushCmd{ConversationID: 1, FromUID: 1, ToUID: 2}); err != nil {
		t.Fatalf("tell typing push: %v", err)
	}
	select {
	case cmd := <-received:
		if cmd.Type != "typing" || cmd.Action != "indicator" {
			t.Fatalf("typing push cmd = %+v", cmd)
		}
	case <-time.After(time.Second):
		t.Fatalf("gateway did not receive typing push")
	}

	// 未知事件和 nil presence 分支都应该安全忽略。
	NewMessagePushActor(nil, nil).handleEvent(nil)
	NewMessagePushActor(presence, nil).handleEvent(unknownPushEvent{})
	NewMessagePushActor(nil, nil).handleTypingPush(conversation.TypingPushCmd{ToUID: 1})
}

func TestMessagePushHandlerMentionPush_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	received := make(chan PushCmd, 10)
	gateway, err := engine.Spawn("push-gateway-mention-test", pushGatewayActor{received: received})
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	presence, err := engine.Spawn("push-presence-mention-test", pushPresenceActor{gateway: gateway})
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	handler, err := engine.Spawn("push-handler-mention-test", NewMessagePushActor(presence, nil))
	if err != nil {
		t.Fatalf("spawn handler: %v", err)
	}

	t.Run("带mention_uids的消息触发mention推送", func(t *testing.T) {
		event := conversation.MessageSentEvent{
			ConversationID: 1,
			MessageID:      100,
			SenderID:       1,
			MemberUIDs:     []uint64{1, 2},
			MentionUIDs:    []uint64{2},
			Content:        "hello",
		}
		if err := handler.Tell(eventbus.EventEnvelope{Event: event}); err != nil {
			t.Fatalf("tell event: %v", err)
		}

		cmds := drainPushCmds(received, 3, time.Second)
		types := map[string]int{}
		for _, cmd := range cmds {
			types[cmd.Action]++
		}
		if types["new"] < 2 {
			t.Fatalf("expected at least 2 'new' pushes, got %d", types["new"])
		}
		if types["mention"] < 1 {
			t.Fatalf("expected at least 1 'mention' push, got %d", types["mention"])
		}
	})

	t.Run("MentionAll触发所有成员mention推送", func(t *testing.T) {
		event := conversation.MessageSentEvent{
			ConversationID: 1,
			MessageID:      200,
			SenderID:       1,
			MemberUIDs:     []uint64{1, 2},
			MentionAll:     true,
			Content:        "@all",
		}
		if err := handler.Tell(eventbus.EventEnvelope{Event: event}); err != nil {
			t.Fatalf("tell event: %v", err)
		}

		cmds := drainPushCmds(received, 3, time.Second)
		mentionCount := 0
		for _, cmd := range cmds {
			if cmd.Action == "mention" {
				mentionCount++
			}
		}
		if mentionCount < 1 {
			t.Fatalf("expected at least 1 mention push for MentionAll, got %d", mentionCount)
		}
	})

	t.Run("无mention不触发mention推送", func(t *testing.T) {
		event := conversation.MessageSentEvent{
			ConversationID: 1,
			MessageID:      300,
			SenderID:       1,
			MemberUIDs:     []uint64{1, 2},
			Content:        "no mention",
		}
		if err := handler.Tell(eventbus.EventEnvelope{Event: event}); err != nil {
			t.Fatalf("tell event: %v", err)
		}

		cmds := drainPushCmds(received, 2, time.Second)
		for _, cmd := range cmds {
			if cmd.Action == "mention" {
				t.Fatalf("should not have mention push for non-mention message")
			}
		}
	})
}

func TestPushMention_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	received := make(chan PushCmd, 10)
	gateway, err := engine.Spawn("push-gateway-mention-unit", pushGatewayActor{received: received})
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	presence, err := engine.Spawn("push-presence-mention-unit", pushPresenceActor{gateway: gateway})
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	handler := NewMessagePushActor(presence, nil)

	t.Run("pushMention跳过发送者", func(t *testing.T) {
		handler.pushMention(conversation.MessageSentEvent{
			ConversationID: 1,
			MessageID:      1,
			SenderID:       1,
			MentionUIDs:    []uint64{1, 2},
		})
		cmds := drainPushCmds(received, 1, time.Second)
		if len(cmds) != 1 {
			t.Fatalf("expected 1 mention push (sender skipped), got %d", len(cmds))
		}
		if cmds[0].Action != "mention" {
			t.Fatalf("expected mention action, got %s", cmds[0].Action)
		}
	})

	t.Run("pushMention MentionAll合并去重", func(t *testing.T) {
		handler.pushMention(conversation.MessageSentEvent{
			ConversationID: 1,
			MessageID:      2,
			SenderID:       3,
			MentionUIDs:    []uint64{1},
			MentionAll:     true,
			MemberUIDs:     []uint64{1, 2, 3},
		})
		cmds := drainPushCmds(received, 2, time.Second)
		if len(cmds) != 2 {
			t.Fatalf("expected 2 mention pushes (sender 3 skipped), got %d", len(cmds))
		}
	})
}

func drainPushCmds(ch chan PushCmd, max int, timeout time.Duration) []PushCmd {
	var cmds []PushCmd
	deadline := time.After(timeout)
	for len(cmds) < max {
		select {
		case cmd := <-ch:
			cmds = append(cmds, cmd)
		case <-deadline:
			return cmds
		}
	}
	return cmds
}

type pushGatewayActor struct {
	received chan PushCmd
}

func (a pushGatewayActor) Receive(ctx actor.Context) {
	if cmd, ok := ctx.Message().(PushCmd); ok {
		a.received <- cmd
	}
}

type pushPresenceActor struct {
	gateway *actor.ActorRef
}

func (a pushPresenceActor) Receive(ctx actor.Context) {
	if query, ok := ctx.Message().(presencedomain.GetGatewaysQuery); ok {
		ctx.Reply(presencedomain.GatewaysResult{UID: query.UID, Gateways: []*actor.ActorRef{a.gateway}})
	}
}

type unknownPushEvent struct{}

func (unknownPushEvent) Name() string { return "unknown" }

func TestRouteCallPushEvents_BitsUT(t *testing.T) {
	tests := []struct {
		name       string
		event      interface{ Name() string }
		wantType   string
		wantAction string
		wantUIDs   int
	}{
		{"来电通知", calldomain.CallIncomingEvent{CalleeUID: 200}, "call", "incoming", 1},
		{"呼叫中", calldomain.CallCallingEvent{CallerUID: 100}, "call", "calling", 1},
		{"已接听", calldomain.CallAcceptedEvent{CallerUID: 100}, "call", "accepted", 1},
		{"已拒绝", calldomain.CallRejectedEvent{CallerUID: 100}, "call", "rejected", 1},
		{"已取消", calldomain.CallCancelledEvent{CalleeUID: 200}, "call", "cancelled", 1},
		{"已结束", calldomain.CallEndedEvent{CallerUID: 100, CalleeUID: 200}, "call", "ended", 2},
		{"超时", calldomain.CallTimeoutEvent{CallerUID: 100, CalleeUID: 200}, "call", "timeout", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pushType, action, recipients, data, ok := routePushEvent(tt.event)
			if !ok {
				t.Fatalf("routePushEvent should match")
			}
			if pushType != tt.wantType || action != tt.wantAction || len(recipients) != tt.wantUIDs || data == nil {
				t.Fatalf("got type=%s action=%s recipients=%v data=%T", pushType, action, recipients, data)
			}
		})
	}
}

func TestCallPushHandlerDeliversToGateway_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	received := make(chan PushCmd, 5)
	gateway, err := engine.Spawn("call-push-gateway", pushGatewayActor{received: received})
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	presence, err := engine.Spawn("call-push-presence", pushPresenceActor{gateway: gateway})
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	handler, err := engine.Spawn("call-push-handler", NewMessagePushActor(presence, nil))
	if err != nil {
		t.Fatalf("spawn handler: %v", err)
	}

	event := calldomain.CallIncomingEvent{
		CallID:       "call_test",
		CallerUID:    100,
		CalleeUID:    200,
		CallType:     1,
		CallerName:   "Alice",
		CallerAvatar: "avatar",
	}
	if err := handler.Tell(eventbus.EventEnvelope{Event: event}); err != nil {
		t.Fatalf("tell event: %v", err)
	}

	select {
	case cmd := <-received:
		if cmd.Type != "call" || cmd.Action != "incoming" {
			t.Fatalf("push cmd = %+v", cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("gateway did not receive incoming push")
	}
}

func TestCallEndedPushesToBoth_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	received := make(chan PushCmd, 5)
	gateway, err := engine.Spawn("call-ended-gateway", pushGatewayActor{received: received})
	if err != nil {
		t.Fatalf("spawn gateway: %v", err)
	}
	presence, err := engine.Spawn("call-ended-presence", pushPresenceActor{gateway: gateway})
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}
	handler, err := engine.Spawn("call-ended-handler", NewMessagePushActor(presence, nil))
	if err != nil {
		t.Fatalf("spawn handler: %v", err)
	}

	event := calldomain.CallEndedEvent{
		CallID:    "call_ended",
		CallerUID: 100,
		CalleeUID: 200,
		StartedAt: 1700000000,
		Duration:  120,
		EndReason: "hangup",
	}
	if err := handler.Tell(eventbus.EventEnvelope{Event: event}); err != nil {
		t.Fatalf("tell event: %v", err)
	}

	cmds := drainPushCmds(received, 2, time.Second)
	if len(cmds) < 2 {
		t.Fatalf("expected at least 2 pushes for ended (both parties), got %d", len(cmds))
	}
	for _, cmd := range cmds {
		if cmd.Type != "call" || cmd.Action != "ended" {
			t.Fatalf("unexpected push: type=%s action=%s", cmd.Type, cmd.Action)
		}
	}
}
