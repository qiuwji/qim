package ws

import (
	"testing"
	"time"

	"qim/internal/actor"
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
		{"正在输入", conversation.TypingEvent{MemberUIDs: []uint64{2}}, "typing", "indicator", 1},
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

	// 未知事件和 nil presence 分支都应该安全忽略。
	NewMessagePushActor(nil, nil).handleEvent(nil, nil)
	NewMessagePushActor(presence, nil).handleEvent(nil, unknownPushEvent{})
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
