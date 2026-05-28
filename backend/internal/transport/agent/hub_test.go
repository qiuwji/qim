package agent

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/eventbus"
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
func (s *testBotStore) ActivateBotTx(creatorUID uint64) (*dal.BotActivationResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &dal.BotActivationResult{BotUser: dal.User{ID: creatorUID, UserType: 1, CreatorUID: creatorUID}}, nil
}

type testAgentStore struct {
	sessions      map[string]*dal.AgentSession
	subscriptions []dal.AgentSubscription
	approvals     map[string]*dal.AgentApproval
	touchCount    map[string]int
	err           error
}

func (s *testAgentStore) CreateSession(session *dal.AgentSession) error {
	if s.err != nil {
		return s.err
	}
	if s.sessions == nil {
		s.sessions = make(map[string]*dal.AgentSession)
	}
	copied := *session
	s.sessions[session.SessionID] = &copied
	return nil
}

func (s *testAgentStore) GetSession(sessionID string) (*dal.AgentSession, error) {
	if s.err != nil {
		return nil, s.err
	}
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	copied := *session
	return &copied, nil
}

func (s *testAgentStore) TouchSession(sessionID string, lastSeenAt, expiresAt int64) error {
	if s.err != nil {
		return s.err
	}
	if session, ok := s.sessions[sessionID]; ok {
		session.LastSeenAt = lastSeenAt
		session.ExpiresAt = expiresAt
		session.UpdatedAt = lastSeenAt
	}
	if s.touchCount == nil {
		s.touchCount = make(map[string]int)
	}
	s.touchCount[sessionID]++
	return nil
}

func (s *testAgentStore) UpsertSubscription(sub *dal.AgentSubscription) error {
	if s.err != nil {
		return s.err
	}
	for i := range s.subscriptions {
		existing := &s.subscriptions[i]
		if existing.SessionID == sub.SessionID && existing.BotUID == sub.BotUID && existing.EventName == sub.EventName && existing.FilterJSON == sub.FilterJSON {
			existing.UpdatedAt = sub.UpdatedAt
			return nil
		}
	}
	s.subscriptions = append(s.subscriptions, *sub)
	return nil
}

func (s *testAgentStore) DeleteSubscriptions(sessionID string, botUID uint64, eventNames []string) error {
	if s.err != nil {
		return s.err
	}
	var filtered []dal.AgentSubscription
	for _, sub := range s.subscriptions {
		if sub.SessionID != sessionID || sub.BotUID != botUID {
			filtered = append(filtered, sub)
			continue
		}
		if len(eventNames) == 0 || containsEvent(eventNames, sub.EventName) {
			continue
		}
		filtered = append(filtered, sub)
	}
	s.subscriptions = filtered
	return nil
}

func (s *testAgentStore) ListSubscriptions() ([]dal.AgentSubscription, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make([]dal.AgentSubscription, len(s.subscriptions))
	copy(result, s.subscriptions)
	return result, nil
}

func (s *testAgentStore) CreateApproval(approval *dal.AgentApproval) error {
	if s.err != nil {
		return s.err
	}
	if s.approvals == nil {
		s.approvals = make(map[string]*dal.AgentApproval)
	}
	copied := *approval
	s.approvals[approval.ApprovalID] = &copied
	return nil
}

func (s *testAgentStore) ResolveApproval(approvalID string, status int8, resolvedAt int64) (*dal.AgentApproval, error) {
	if s.err != nil {
		return nil, s.err
	}
	approval, ok := s.approvals[approvalID]
	if !ok || approval.Status != dal.AgentApprovalStatusPending {
		return nil, errors.New("approval not found")
	}
	approval.Status = status
	approval.ResolvedAt = resolvedAt
	approval.UpdatedAt = resolvedAt
	copied := *approval
	return &copied, nil
}

func (s *testAgentStore) ListPendingApprovals() ([]dal.AgentApproval, error) {
	if s.err != nil {
		return nil, s.err
	}
	var result []dal.AgentApproval
	for _, approval := range s.approvals {
		if approval.Status == dal.AgentApprovalStatusPending && approval.ExpiresAt > time.Now().Unix() {
			result = append(result, *approval)
		}
	}
	return result, nil
}

func (s *testAgentStore) GetApproval(approvalID string) (*dal.AgentApproval, error) {
	if s.err != nil {
		return nil, s.err
	}
	approval, ok := s.approvals[approvalID]
	if !ok {
		return nil, errors.New("approval not found")
	}
	copied := *approval
	return &copied, nil
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func containsEvent(events []string, target string) bool {
	for _, event := range events {
		if event == target {
			return true
		}
	}
	return false
}

type testEventBus struct{}

func (testEventBus) Publish(eventbus.Event) error {
	return nil
}

func (testEventBus) Subscribe(string, *actor.ActorRef) error {
	return nil
}

func (testEventBus) Unsubscribe(string, *actor.ActorRef) error {
	return nil
}

type testAgentGatewayProbe struct {
	events chan PushNotificationCmd
}

func (p *testAgentGatewayProbe) Receive(ctx actor.Context) {
	if msg, ok := ctx.Message().(PushNotificationCmd); ok {
		p.events <- msg
	}
}

func TestAgentHubActor_SubscribeUnsubscribe_BitsUT(t *testing.T) {
	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	agentStore := &testAgentStore{}
	ref := spawnTestHub(t, "agent-hub-subscribe", store, agentStore)

	if err := ref.Tell(SubscribeCmd{SessionID: "s1", BotUID: 100, Events: []string{"message_sent"}}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SubscribeCmd{SessionID: "s1", BotUID: 100, Events: []string{"message_revoked"}}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)

	if len(agentStore.subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(agentStore.subscriptions))
	}

	if err := ref.Tell(UnsubscribeCmd{SessionID: "s1", BotUID: 100, Events: []string{"message_sent"}}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)
	if containsStoredSubscription(agentStore, "s1", 100, conversation.EventMessageSent) {
		t.Fatal("expected message_sent removed")
	}

	if err := ref.Tell(UnsubscribeCmd{SessionID: "s1", BotUID: 100}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)
	if len(agentStore.subscriptions) != 0 {
		t.Fatalf("expected all removed, got %d", len(agentStore.subscriptions))
	}
}

func TestAgentHubActor_EchoSuppression_BitsUT(t *testing.T) {
	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	ref := spawnTestHub(t, "agent-hub-echo", store, &testAgentStore{})
	probeRef, probe := spawnGatewayProbe(t, "agent-hub-echo-gw")

	if err := ref.Tell(AgentRefResolved{GwRef: probeRef}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SubscribeCmd{SessionID: "s1", BotUID: 100, Events: []string{"message_sent"}}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)

	if err := ref.Tell(eventbus.EventEnvelope{Event: conversation.MessageSentEvent{SenderID: 100, ConversationID: 1}}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)
	assertNoGatewayPush(t, probe.events)

	if err := ref.Tell(eventbus.EventEnvelope{Event: conversation.MessageSentEvent{SenderID: 50, ConversationID: 1}}); err != nil {
		t.Fatal(err)
	}
	got := waitGatewayPush(t, probe.events)
	if got.BotUID != 100 || got.SessionID != "s1" || got.Type != "message_sent" {
		t.Fatalf("unexpected push: %+v", got)
	}
}

func TestAgentHubActor_FilterByConvID_BitsUT(t *testing.T) {
	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	ref := spawnTestHub(t, "agent-hub-filter", store, &testAgentStore{})
	probeRef, probe := spawnGatewayProbe(t, "agent-hub-filter-gw")

	conv5 := uint64(5)
	if err := ref.Tell(AgentRefResolved{GwRef: probeRef}); err != nil {
		t.Fatal(err)
	}
	if err := ref.Tell(SubscribeCmd{
		SessionID: "s1", BotUID: 100, Events: []string{"message_sent"},
		Filter: &EventFilter{ConvID: &conv5},
	}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)

	if err := ref.Tell(eventbus.EventEnvelope{Event: conversation.MessageSentEvent{ConversationID: 99, SenderID: 50}}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)
	assertNoGatewayPush(t, probe.events)

	if err := ref.Tell(eventbus.EventEnvelope{Event: conversation.MessageSentEvent{ConversationID: 5, SenderID: 50}}); err != nil {
		t.Fatal(err)
	}
	got := waitGatewayPush(t, probe.events)
	if got.BotUID != 100 || got.SessionID != "s1" || got.Type != "message_sent" {
		t.Fatalf("unexpected push: %+v", got)
	}
}

func TestAgentHubActor_PermissionCache_BitsUT(t *testing.T) {
	store := &testBotStore{
		configs: map[uint64]*dal.BotConfig{
			100: {UID: 100, Permissions: `["friend:read","message:search"]`},
		},
	}
	ref := spawnTestHub(t, "agent-hub-permission", store, &testAgentStore{})
	if err := ref.Tell(RefreshPermissionsCmd{BotUID: 100}); err != nil {
		t.Fatal(err)
	}
	waitHubIdle(t, ref)

	if !askPermission(t, ref, 100, "friend:read") {
		t.Fatal("expected friend:read")
	}
	if askPermission(t, ref, 100, "group:read") {
		t.Fatal("expected no group:read")
	}
	if askPermission(t, ref, 999, "friend:read") {
		t.Fatal("expected false for unknown bot")
	}
}

func spawnTestHub(t *testing.T, name string, botStore dal.BotStore, agentStore dal.AgentStore) *actor.ActorRef {
	t.Helper()
	engine := actor.NewEngine()
	ref, err := engine.Spawn(name, NewAgentHubActor(testEventBus{}, botStore, agentStore))
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func spawnGatewayProbe(t *testing.T, name string) (*actor.ActorRef, *testAgentGatewayProbe) {
	t.Helper()
	engine := actor.NewEngine()
	probe := &testAgentGatewayProbe{events: make(chan PushNotificationCmd, 8)}
	ref, err := engine.Spawn(name, probe)
	if err != nil {
		t.Fatal(err)
	}
	return ref, probe
}

func waitHubIdle(t *testing.T, ref *actor.ActorRef) {
	t.Helper()
	_ = askPermission(t, ref, 0, "__barrier__")
}

func askPermission(t *testing.T, ref *actor.ActorRef, botUID uint64, permission string) bool {
	t.Helper()
	raw, err := ref.Ask(PermissionQuery{BotUID: botUID, Permission: permission}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	allowed, ok := raw.(bool)
	if !ok {
		t.Fatalf("expected bool permission response, got %T", raw)
	}
	return allowed
}

func waitGatewayPush(t *testing.T, ch <-chan PushNotificationCmd) PushNotificationCmd {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("expected gateway push")
	}
	return PushNotificationCmd{}
}

func assertNoGatewayPush(t *testing.T, ch <-chan PushNotificationCmd) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("unexpected gateway push: %+v", msg)
	case <-time.After(20 * time.Millisecond):
	}
}

func containsStoredSubscription(store *testAgentStore, sessionID string, botUID uint64, eventName string) bool {
	for _, sub := range store.subscriptions {
		if sub.SessionID == sessionID && sub.BotUID == botUID && sub.EventName == eventName {
			return true
		}
	}
	return false
}
