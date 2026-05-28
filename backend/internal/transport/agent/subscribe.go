package agent

import (
	"fmt"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
)

type SubscribeRouter struct {
	hubRef *actor.ActorRef
	bots   dal.BotStore
}

var validEvents = map[string]bool{
	"message_sent":         true,
	"message_revoked":      true,
	"member_joined":        true,
	"member_left":          true,
	"conversation_updated": true,
}

func NewSubscribeRouter(hubRef *actor.ActorRef, botStore ...dal.BotStore) *SubscribeRouter {
	var bots dal.BotStore
	if len(botStore) > 0 {
		bots = botStore[0]
	}
	return &SubscribeRouter{hubRef: hubRef, bots: bots}
}

type subscribeParams struct {
	BotUID uint64   `json:"bot_uid"`
	Events []string `json:"events"`
	Filter *struct {
		ConvID *uint64 `json:"conv_id"`
	} `json:"filter,omitempty"`
}

type unsubscribeParams struct {
	BotUID uint64   `json:"bot_uid"`
	Events []string `json:"events"`
}

func (r *SubscribeRouter) Resolve(sessionID string, action string, params any) (any, error) {
	switch action {
	case "subscribe_events":
		p, ok := params.(subscribeParams)
		if !ok {
			return nil, fmt.Errorf("invalid subscribe params")
		}
		return r.subscribe(sessionID, p)
	case "unsubscribe_events":
		p, ok := params.(unsubscribeParams)
		if !ok {
			return nil, fmt.Errorf("invalid unsubscribe params")
		}
		return r.unsubscribe(sessionID, p)
	default:
		return nil, fmt.Errorf("unknown subscribe action: %s", action)
	}
}

func (r *SubscribeRouter) subscribe(sessionID string, p subscribeParams) (any, error) {
	if err := r.validateBot(p.BotUID); err != nil {
		return nil, err
	}
	events := make([]string, 0, len(p.Events))
	for _, evt := range p.Events {
		if !validEvents[evt] {
			return nil, ErrInvalidEvent
		}
		events = append(events, agentEventToBusEvent(evt))
	}
	var filter *EventFilter
	if p.Filter != nil {
		filter = &EventFilter{ConvID: p.Filter.ConvID}
	}
	if err := r.hubRef.Tell(SubscribeCmd{SessionID: sessionID, BotUID: p.BotUID, Events: events, Filter: filter}); err != nil {
		return nil, err
	}
	return map[string]string{"status": "ok"}, nil
}

func (r *SubscribeRouter) unsubscribe(sessionID string, p unsubscribeParams) (any, error) {
	if err := r.validateBot(p.BotUID); err != nil {
		return nil, err
	}
	events := make([]string, 0, len(p.Events))
	for _, evt := range p.Events {
		events = append(events, agentEventToBusEvent(evt))
	}
	if err := r.hubRef.Tell(UnsubscribeCmd{SessionID: sessionID, BotUID: p.BotUID, Events: events}); err != nil {
		return nil, err
	}
	return map[string]string{"status": "ok"}, nil
}

func (r *SubscribeRouter) validateBot(botUID uint64) error {
	if r.bots == nil {
		return nil
	}
	cfg, err := r.bots.GetConfig(botUID)
	if err != nil || cfg == nil {
		return ErrBotNotFound
	}
	return nil
}

func agentEventToBusEvent(eventName string) string {
	switch eventName {
	case "message_sent", conversation.EventMessageSent:
		return conversation.EventMessageSent
	case "message_revoked", conversation.EventMessageRevoked:
		return conversation.EventMessageRevoked
	case "member_joined", conversation.EventMemberJoined:
		return conversation.EventMemberJoined
	case "member_left", conversation.EventMemberLeft:
		return conversation.EventMemberLeft
	case "conversation_updated", conversation.EventConversationUpdated:
		return conversation.EventConversationUpdated
	default:
		return eventName
	}
}
