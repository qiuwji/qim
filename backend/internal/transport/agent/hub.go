package agent

import (
	"encoding/json"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/eventbus"

	"go.uber.org/zap"
)

type subscription struct {
	eventName string
	filter    *EventFilter
}

type AgentHubActor struct {
	events        eventbus.Bus
	botStore      dal.BotStore
	subscriptions map[string]map[uint64][]subscription
	permCache     map[uint64][]string
	gwRef         *actor.ActorRef
}

func NewAgentHubActor(events eventbus.Bus, botStore dal.BotStore) *AgentHubActor {
	return &AgentHubActor{
		events:        events,
		botStore:      botStore,
		subscriptions: make(map[string]map[uint64][]subscription),
		permCache:     make(map[uint64][]string),
	}
}

func (h *AgentHubActor) OnStart(ctx actor.Context) {
	if err := h.loadPermissionCache(); err != nil {
		zap.L().Error("agent hub load permission cache failed", zap.Error(err))
	}
	eventNames := []string{
		conversation.EventMessageSent,
		conversation.EventMessageRevoked,
		conversation.EventMemberJoined,
		conversation.EventMemberLeft,
		conversation.EventConversationUpdated,
	}
	for _, name := range eventNames {
		if err := h.events.Subscribe(name, ctx.Self()); err != nil {
			zap.L().Error("agent hub subscribe event failed", zap.String("event", name), zap.Error(err))
		}
	}
	zap.L().Info("AgentHubActor started", zap.Int("event_types", len(eventNames)))
}

func (h *AgentHubActor) loadPermissionCache() error {
	configs, err := h.botStore.ListAllConfigs()
	if err != nil {
		return err
	}
	for _, cfg := range configs {
		h.permCache[cfg.UID] = parsePermissions(cfg.Permissions)
	}
	return nil
}

func (h *AgentHubActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case eventbus.EventEnvelope:
		h.handleEvent(msg.Event)
	case SubscribeCmd:
		h.handleSubscribe(msg, ctx)
	case UnsubscribeCmd:
		h.handleUnsubscribe(msg)
	case RefreshPermissionsCmd:
		h.refreshPermissions(msg.BotUID, ctx)
	case AgentRefResolved:
		h.gwRef = msg.GwRef
	}
}

func (h *AgentHubActor) handleEvent(event eventbus.Event) {
	eventName := event.Name()
	subs := h.subscriptions[eventName]
	if subs == nil {
		return
	}
	for botUID, botSubs := range subs {
		for _, sub := range botSubs {
			if sub.filter != nil && !sub.filter.Match(event) {
				continue
			}
			if e, ok := event.(conversation.MessageSentEvent); ok {
				if e.SenderID == botUID {
					continue
				}
			}
			if h.gwRef != nil {
				h.gwRef.Tell(PushNotificationCmd{
					BotUID: botUID,
					Type:   toAgentEventName(eventName),
					Event:  event,
				})
			}
		}
	}
}

func (h *AgentHubActor) handleSubscribe(cmd SubscribeCmd, ctx actor.Context) {
	for _, eventName := range cmd.Events {
		if h.subscriptions[eventName] == nil {
			h.subscriptions[eventName] = make(map[uint64][]subscription)
		}
		h.subscriptions[eventName][cmd.BotUID] = append(
			h.subscriptions[eventName][cmd.BotUID],
			subscription{eventName: eventName, filter: cmd.Filter},
		)
	}
}

func (h *AgentHubActor) handleUnsubscribe(cmd UnsubscribeCmd) {
	if len(cmd.Events) == 0 {
		for eventName := range h.subscriptions {
			delete(h.subscriptions[eventName], cmd.BotUID)
			if len(h.subscriptions[eventName]) == 0 {
				delete(h.subscriptions, eventName)
			}
		}
		return
	}
	for _, eventName := range cmd.Events {
		if h.subscriptions[eventName] != nil {
			delete(h.subscriptions[eventName], cmd.BotUID)
			if len(h.subscriptions[eventName]) == 0 {
				delete(h.subscriptions, eventName)
			}
		}
	}
}

func (h *AgentHubActor) refreshPermissions(botUID uint64, _ actor.Context) {
	cfg, err := h.botStore.GetConfig(botUID)
	if err != nil {
		delete(h.permCache, botUID)
		return
	}
	h.permCache[botUID] = parsePermissions(cfg.Permissions)
}

func (h *AgentHubActor) HasPermission(botUID uint64, perm string) bool {
	perms, ok := h.permCache[botUID]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

func (s *EventFilter) Match(event eventbus.Event) bool {
	if s == nil || s.ConvID == nil {
		return true
	}
	switch e := event.(type) {
	case conversation.MessageSentEvent:
		return e.ConversationID == *s.ConvID
	case conversation.MessageRevokedEvent:
		return e.ConversationID == *s.ConvID
	case conversation.MemberJoinedEvent:
		return e.ConversationID == *s.ConvID
	case conversation.MemberLeftEvent:
		return e.ConversationID == *s.ConvID
	case conversation.ConversationUpdatedEvent:
		return e.ConversationID == *s.ConvID
	}
	return false
}

func toAgentEventName(eventName string) string {
	switch eventName {
	case conversation.EventMessageSent:
		return "message_sent"
	case conversation.EventMessageRevoked:
		return "message_revoked"
	case conversation.EventMemberJoined:
		return "member_joined"
	case conversation.EventMemberLeft:
		return "member_left"
	case conversation.EventConversationUpdated:
		return "conversation_updated"
	default:
		return eventName
	}
}

func parsePermissions(data string) []string {
	if data == "" || data == "[]" {
		return nil
	}
	var perms []string
	if err := json.Unmarshal([]byte(data), &perms); err != nil {
		return nil
	}
	return perms
}
