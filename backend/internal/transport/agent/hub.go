package agent

import (
	"encoding/json"
	"time"

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
	agentStore    dal.AgentStore
	subscriptions map[string]map[string]map[uint64][]subscription
	permCache     map[uint64][]string
	gwRef         *actor.ActorRef
}

func NewAgentHubActor(events eventbus.Bus, botStore dal.BotStore, agentStore dal.AgentStore) *AgentHubActor {
	return &AgentHubActor{
		events:        events,
		botStore:      botStore,
		agentStore:    agentStore,
		subscriptions: make(map[string]map[string]map[uint64][]subscription),
		permCache:     make(map[uint64][]string),
	}
}

func (h *AgentHubActor) OnStart(ctx actor.Context) {
	if err := h.loadPermissionCache(); err != nil {
		zap.L().Error("agent hub load permission cache failed", zap.Error(err))
	}
	if err := h.loadSubscriptions(); err != nil {
		zap.L().Error("agent hub load subscriptions failed", zap.Error(err))
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
	case PermissionQuery:
		_ = ctx.Reply(h.hasPermission(msg.BotUID, msg.Permission))
	}
}

func (h *AgentHubActor) handleEvent(event eventbus.Event) {
	eventName := event.Name()
	subs := h.subscriptions[eventName]
	if subs == nil {
		return
	}
	for sessionID, sessionSubs := range subs {
		for botUID, botSubs := range sessionSubs {
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
						SessionID: sessionID,
						BotUID:    botUID,
						Type:      toAgentEventName(eventName),
						Event:     event,
					})
				}
			}
		}
	}
}

func (h *AgentHubActor) handleSubscribe(cmd SubscribeCmd, ctx actor.Context) {
	for _, rawEventName := range cmd.Events {
		eventName := agentEventToBusEvent(rawEventName)
		if h.agentStore != nil {
			_ = h.agentStore.UpsertSubscription(&dal.AgentSubscription{
				SessionID:  cmd.SessionID,
				BotUID:     cmd.BotUID,
				EventName:  eventName,
				FilterJSON: encodeFilter(cmd.Filter),
				CreatedAt:  timeNowUnix(),
				UpdatedAt:  timeNowUnix(),
			})
		}
		if h.subscriptions[eventName] == nil {
			h.subscriptions[eventName] = make(map[string]map[uint64][]subscription)
		}
		if h.subscriptions[eventName][cmd.SessionID] == nil {
			h.subscriptions[eventName][cmd.SessionID] = make(map[uint64][]subscription)
		}
		h.subscriptions[eventName][cmd.SessionID][cmd.BotUID] = append(
			h.subscriptions[eventName][cmd.SessionID][cmd.BotUID],
			subscription{eventName: eventName, filter: cmd.Filter},
		)
	}
}

func (h *AgentHubActor) handleUnsubscribe(cmd UnsubscribeCmd) {
	if h.agentStore != nil {
		_ = h.agentStore.DeleteSubscriptions(cmd.SessionID, cmd.BotUID, toBusEventNames(cmd.Events))
	}
	if len(cmd.Events) == 0 {
		for eventName := range h.subscriptions {
			if h.subscriptions[eventName][cmd.SessionID] != nil {
				delete(h.subscriptions[eventName][cmd.SessionID], cmd.BotUID)
				if len(h.subscriptions[eventName][cmd.SessionID]) == 0 {
					delete(h.subscriptions[eventName], cmd.SessionID)
				}
			}
			if len(h.subscriptions[eventName]) == 0 {
				delete(h.subscriptions, eventName)
			}
		}
		return
	}
	for _, rawEventName := range cmd.Events {
		eventName := agentEventToBusEvent(rawEventName)
		if h.subscriptions[eventName] != nil {
			if h.subscriptions[eventName][cmd.SessionID] != nil {
				delete(h.subscriptions[eventName][cmd.SessionID], cmd.BotUID)
				if len(h.subscriptions[eventName][cmd.SessionID]) == 0 {
					delete(h.subscriptions[eventName], cmd.SessionID)
				}
			}
			if len(h.subscriptions[eventName]) == 0 {
				delete(h.subscriptions, eventName)
			}
		}
	}
}

func (h *AgentHubActor) loadSubscriptions() error {
	if h.agentStore == nil {
		return nil
	}
	subs, err := h.agentStore.ListSubscriptions()
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if h.subscriptions[sub.EventName] == nil {
			h.subscriptions[sub.EventName] = make(map[string]map[uint64][]subscription)
		}
		if h.subscriptions[sub.EventName][sub.SessionID] == nil {
			h.subscriptions[sub.EventName][sub.SessionID] = make(map[uint64][]subscription)
		}
		h.subscriptions[sub.EventName][sub.SessionID][sub.BotUID] = append(
			h.subscriptions[sub.EventName][sub.SessionID][sub.BotUID],
			subscription{eventName: sub.EventName, filter: decodeFilter(sub.FilterJSON)},
		)
	}
	return nil
}

func (h *AgentHubActor) refreshPermissions(botUID uint64, _ actor.Context) {
	cfg, err := h.botStore.GetConfig(botUID)
	if err != nil || cfg == nil {
		delete(h.permCache, botUID)
		return
	}
	h.permCache[botUID] = parsePermissions(cfg.Permissions)
}

func toBusEventNames(events []string) []string {
	if len(events) == 0 {
		return nil
	}
	result := make([]string, 0, len(events))
	for _, event := range events {
		result = append(result, agentEventToBusEvent(event))
	}
	return result
}

func (h *AgentHubActor) hasPermission(botUID uint64, perm string) bool {
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

func encodeFilter(filter *EventFilter) string {
	if filter == nil {
		return ""
	}
	b, err := json.Marshal(filter)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeFilter(raw string) *EventFilter {
	if raw == "" {
		return nil
	}
	var filter EventFilter
	if err := json.Unmarshal([]byte(raw), &filter); err != nil {
		return nil
	}
	return &filter
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
