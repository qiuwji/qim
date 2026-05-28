package dal

import (
	"testing"
	"time"
)

func TestAgentStoreSession_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewAgentStore(db)

	session := NewAgentSession("session-1", time.Hour)
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	got, err := store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if got.SessionID != "session-1" || got.Status != AgentSessionStatusActive {
		t.Fatalf("unexpected session: %+v", got)
	}

	now := time.Now().Unix()
	if err := store.TouchSession("session-1", now, now+3600); err != nil {
		t.Fatalf("TouchSession error: %v", err)
	}
	got, err = store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession after touch error: %v", err)
	}
	if got.LastSeenAt != now {
		t.Fatalf("LastSeenAt = %d, want %d", got.LastSeenAt, now)
	}
}

func TestAgentStoreSubscription_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewAgentStore(db)
	now := time.Now().Unix()

	sub := &AgentSubscription{
		SessionID:  "session-1",
		BotUID:     100,
		EventName:  "conversation.message_sent",
		FilterJSON: `{"ConvID":5}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.UpsertSubscription(sub); err != nil {
		t.Fatalf("UpsertSubscription error: %v", err)
	}
	if err := store.UpsertSubscription(sub); err != nil {
		t.Fatalf("UpsertSubscription second error: %v", err)
	}

	subs, err := store.ListSubscriptions()
	if err != nil {
		t.Fatalf("ListSubscriptions error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("subscriptions len = %d, want 1", len(subs))
	}

	if err := store.DeleteSubscriptions("session-1", 100, []string{"conversation.message_sent"}); err != nil {
		t.Fatalf("DeleteSubscriptions error: %v", err)
	}
	subs, err = store.ListSubscriptions()
	if err != nil {
		t.Fatalf("ListSubscriptions after delete error: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("subscriptions len after delete = %d, want 0", len(subs))
	}
}

func TestAgentStoreApproval_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewAgentStore(db)
	now := time.Now().Unix()

	approval := &AgentApproval{
		ApprovalID:     "approval-1",
		SessionID:      "session-1",
		BotUID:         100,
		OwnerUID:       1,
		ConversationID: 10,
		Action:         "send_message",
		Detail:         "detail",
		Status:         AgentApprovalStatusPending,
		ExpiresAt:      now + 60,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.CreateApproval(approval); err != nil {
		t.Fatalf("CreateApproval error: %v", err)
	}

	pending, err := store.ListPendingApprovals()
	if err != nil {
		t.Fatalf("ListPendingApprovals error: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending approvals len = %d, want 1", len(pending))
	}

	resolved, err := store.ResolveApproval("approval-1", AgentApprovalStatusApproved, now+1)
	if err != nil {
		t.Fatalf("ResolveApproval error: %v", err)
	}
	if resolved.Status != AgentApprovalStatusApproved {
		t.Fatalf("resolved status = %d, want %d", resolved.Status, AgentApprovalStatusApproved)
	}

	got, err := store.GetApproval("approval-1")
	if err != nil {
		t.Fatalf("GetApproval error: %v", err)
	}
	if got.Status != AgentApprovalStatusApproved {
		t.Fatalf("stored status = %d, want %d", got.Status, AgentApprovalStatusApproved)
	}
}
