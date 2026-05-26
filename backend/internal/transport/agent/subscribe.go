package agent

import (
	"fmt"

	"qim/internal/actor"
)

type SubscribeRouter struct {
	hubRef *actor.ActorRef
}

var validEvents = map[string]bool{
	"message_sent":          true,
	"message_revoked":       true,
	"member_joined":         true,
	"member_left":           true,
	"conversation_updated":  true,
}

func NewSubscribeRouter(hubRef *actor.ActorRef) *SubscribeRouter {
	return &SubscribeRouter{hubRef: hubRef}
}

type subscribeParams struct {
	BotUID uint64  `json:"bot_uid"`
	Events []string `json:"events"`
	Filter *struct {
		ConvID *uint64 `json:"conv_id"`
	} `json:"filter,omitempty"`
}

type unsubscribeParams struct {
	BotUID uint64  `json:"bot_uid"`
	Events []string `json:"events"`
}

func (r *SubscribeRouter) Resolve(action string, params any) (any, error) {
	switch action {
	case "subscribe_events":
		p, ok := params.(subscribeParams)
		if !ok {
			return nil, fmt.Errorf("invalid subscribe params")
		}
		return r.subscribe(p)
	case "unsubscribe_events":
		p, ok := params.(unsubscribeParams)
		if !ok {
			return nil, fmt.Errorf("invalid unsubscribe params")
		}
		return r.unsubscribe(p)
	default:
		return nil, fmt.Errorf("unknown subscribe action: %s", action)
	}
}

func (r *SubscribeRouter) subscribe(p subscribeParams) (any, error) {
	for _, evt := range p.Events {
		if !validEvents[evt] {
			return nil, ErrInvalidEvent
		}
	}
	var filter *EventFilter
	if p.Filter != nil {
		filter = &EventFilter{ConvID: p.Filter.ConvID}
	}
	if err := r.hubRef.Tell(SubscribeCmd{BotUID: p.BotUID, Events: p.Events, Filter: filter}); err != nil {
		return nil, err
	}
	return map[string]string{"status": "ok"}, nil
}

func (r *SubscribeRouter) unsubscribe(p unsubscribeParams) (any, error) {
	if err := r.hubRef.Tell(UnsubscribeCmd{BotUID: p.BotUID, Events: p.Events}); err != nil {
		return nil, err
	}
	return map[string]string{"status": "ok"}, nil
}
