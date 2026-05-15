package ws

import (
	"context"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	presencedomain "qim/internal/domain/presence"
	"qim/internal/eventbus"
)

const presenceAskTimeout = time.Second

type MessagePushHandler struct {
	presence *actor.ActorRef
}

func NewMessagePushHandler(presence *actor.ActorRef) *MessagePushHandler {
	return &MessagePushHandler{presence: presence}
}

func (h *MessagePushHandler) Handle(ctx context.Context, event eventbus.Event) error {
	e, ok := event.(conversation.MessageSentEvent)
	if !ok || h.presence == nil {
		return nil
	}

	for _, uid := range uniqueUIDs(e.MemberUIDs) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		h.pushToUser(uid, e)
	}
	return nil
}

func (h *MessagePushHandler) pushToUser(uid uint64, event conversation.MessageSentEvent) {
	raw, err := h.presence.Ask(presencedomain.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		return
	}
	result, ok := raw.(presencedomain.GatewaysResult)
	if !ok {
		return
	}
	for _, gateway := range result.Gateways {
		_ = gateway.Tell(PushCmd{Type: "message", Data: event})
	}
}

func uniqueUIDs(uids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(uids))
	result := make([]uint64, 0, len(uids))
	for _, uid := range uids {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		result = append(result, uid)
	}
	return result
}
