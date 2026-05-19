package dal

import (
	"testing"

	"qim/internal/domain/call"
)

func TestCallStore_Create_BitsUT(t *testing.T) {
	db := newTestDB(t)
	store := NewCallStore(db)

	record := &call.Call{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  call.CallTypeVoice,
		Status:    call.CallStatusCompleted,
		StartedAt: 1700000000,
		EndedAt:   1700000120,
		Duration:  120,
		EndReason: "hangup",
		CreatedAt: 1700000120,
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.ID == 0 {
		t.Fatal("expected auto-increment ID")
	}
}

func TestCallStore_CreateMissedCall_BitsUT(t *testing.T) {
	db := newTestDB(t)
	store := NewCallStore(db)

	record := &call.Call{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  call.CallTypeVoice,
		Status:    call.CallStatusMissed,
		StartedAt: 0,
		EndedAt:   1700000000,
		Duration:  0,
		EndReason: "timeout",
		CreatedAt: 1700000000,
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.Duration != 0 {
		t.Fatalf("expected duration 0, got %d", record.Duration)
	}
}

func TestCallStore_CreateVideoCall_BitsUT(t *testing.T) {
	db := newTestDB(t)
	store := NewCallStore(db)

	record := &call.Call{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  call.CallTypeVideo,
		Status:    call.CallStatusCompleted,
		StartedAt: 1700000000,
		EndedAt:   1700000045,
		Duration:  45,
		EndReason: "hangup",
		CreatedAt: 1700000045,
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.CallType != call.CallTypeVideo {
		t.Fatal("expected video call type")
	}
}
