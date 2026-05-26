package agent

import (
	"encoding/json"
	"testing"

	"qim/internal/dal"
	"qim/internal/domain/conversation"
)

type testBotStore struct {
	configs map[uint64]*dal.BotConfig
	err     error
}

func (s *testBotStore) GetConfig(uid uint64) (*dal.BotConfig, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.configs[uid], nil
}
func (s *testBotStore) CreateConfig(config *dal.BotConfig) error {
	if s.err != nil {
		return s.err
	}
	s.configs[config.UID] = config
	return nil
}
func (s *testBotStore) UpdatePermissions(uid uint64, permissions []string) error {
	if s.err != nil {
		return s.err
	}
	if cfg, ok := s.configs[uid]; ok {
		cfg.Permissions = toJSON(permissions)
	}
	return nil
}
func (s *testBotStore) ListAllConfigs() ([]dal.BotConfig, error) {
	if s.err != nil {
		return nil, s.err
	}
	var result []dal.BotConfig
	for _, cfg := range s.configs {
		result = append(result, *cfg)
	}
	return result, nil
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestAgentHubActor_SubscribeUnsubscribe_BitsUT(t *testing.T) {
	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	hub := NewAgentHubActor(nil, store)

	hub.handleSubscribe(SubscribeCmd{BotUID: 100, Events: []string{"message_sent"}}, nil)
	hub.handleSubscribe(SubscribeCmd{BotUID: 100, Events: []string{"message_revoked"}}, nil)

	if len(hub.subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(hub.subscriptions))
	}

	hub.handleUnsubscribe(UnsubscribeCmd{BotUID: 100, Events: []string{"message_sent"}})
	if _, ok := hub.subscriptions[conversation.EventMessageSent]; ok {
		t.Fatal("expected message_sent removed")
	}

	hub.handleUnsubscribe(UnsubscribeCmd{BotUID: 100})
	if len(hub.subscriptions) != 0 {
		t.Fatalf("expected all removed, got %d", len(hub.subscriptions))
	}
}

func TestAgentHubActor_EchoSuppression_BitsUT(t *testing.T) {
	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	hub := NewAgentHubActor(nil, store)

	hub.handleSubscribe(SubscribeCmd{BotUID: 100, Events: []string{"message_sent"}}, nil)

	sentByBot := conversation.MessageSentEvent{SenderID: 100, ConversationID: 1}
	sentByOther := conversation.MessageSentEvent{SenderID: 50, ConversationID: 1}

	if hub.subscriptions["message_sent"][100] == nil {
		t.Fatal("subscription not found")
	}
	if sentByBot.SenderID != 100 {
		t.Fatal("SenderID mismatch for bot message")
	}
	if sentByOther.SenderID == 100 {
		t.Fatal("false suppression: other's msg should not be suppressed")
	}
}

func TestAgentHubActor_FilterByConvID_BitsUT(t *testing.T) {
	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	hub := NewAgentHubActor(nil, store)

	conv5 := uint64(5)
	hub.handleSubscribe(SubscribeCmd{
		BotUID: 100, Events: []string{"message_sent"},
		Filter: &EventFilter{ConvID: &conv5},
	}, nil)

	subs := hub.subscriptions["message_sent"]
	if subs == nil || len(subs[100]) != 1 {
		t.Fatal("subscription not registered")
	}
	sub := subs[100][0]
	if sub.filter == nil || sub.filter.ConvID == nil || *sub.filter.ConvID != 5 {
		t.Fatal("filter conv_id not set correctly")
	}

	if sub.filter.Match(conversation.MessageSentEvent{ConversationID: 5}) != true {
		t.Fatal("filter should match conv 5")
	}
	if sub.filter.Match(conversation.MessageSentEvent{ConversationID: 99}) != false {
		t.Fatal("filter should not match conv 99")
	}
}

func TestAgentHubActor_PermissionCache_BitsUT(t *testing.T) {
	store := &testBotStore{
		configs: map[uint64]*dal.BotConfig{
			100: {UID: 100, Permissions: `["friend:read","message:search"]`},
		},
	}
	hub := NewAgentHubActor(nil, store)
	hub.loadPermissionCache()

	if !hub.HasPermission(100, "friend:read") {
		t.Fatal("expected friend:read")
	}
	if hub.HasPermission(100, "group:read") {
		t.Fatal("expected no group:read")
	}
	if hub.HasPermission(999, "friend:read") {
		t.Fatal("expected false for unknown bot")
	}
}
