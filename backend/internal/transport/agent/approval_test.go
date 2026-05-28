package agent

import (
	"testing"
	"time"

	"qim/internal/dal"
)

func TestApprovalManager_Approve_BitsUT(t *testing.T) {
	store := &testAgentStore{approvals: make(map[string]*dal.AgentApproval)}
	mgr := NewApprovalManager(store)
	mgr.newID = func() string { return "approval-1" }
	done := make(chan map[string]string, 1)
	errs := make(chan error, 1)

	go func() {
		res, err := mgr.RequestUserApprovalAndWait("session-1", nil, 1, 100, 10, "send_message", "test", 5)
		if err != nil {
			errs <- err
			return
		}
		done <- res
	}()

	requireApprovalResolved(t, mgr, "approval-1", "approved")
	select {
	case err := <-errs:
		t.Fatalf("Request error: %v", err)
	case res := <-done:
		if res["status"] != "approved" || res["approval_id"] != "approval-1" {
			t.Fatalf("unexpected result: %v", res)
		}
	}
	if mgr.ResolveApproval("approval-1", "approved") {
		t.Fatal("expected resolved approval id to be rejected")
	}
	approval, err := store.GetApproval("approval-1")
	if err != nil {
		t.Fatalf("GetApproval error: %v", err)
	}
	if approval.Status != dal.AgentApprovalStatusApproved {
		t.Fatalf("expected approval status approved, got %d", approval.Status)
	}
}

func TestApprovalManager_Timeout_BitsUT(t *testing.T) {
	mgr := NewApprovalManager(&testAgentStore{approvals: make(map[string]*dal.AgentApproval)})
	res, err := mgr.RequestUserApprovalAndWait("session-1", nil, 1, 100, 10, "send_message", "test", 1)
	if err != nil {
		t.Fatalf("Request error: %v", err)
	}
	if res["status"] != "timeout" || res["approval_id"] == "" {
		t.Fatalf("unexpected timeout result: %v", res)
	}
}

func TestApprovalManager_InvalidResolve_BitsUT(t *testing.T) {
	mgr := NewApprovalManager(&testAgentStore{approvals: make(map[string]*dal.AgentApproval)})
	mgr.newID = func() string { return "approval-invalid" }
	errs := make(chan error, 1)

	go func() {
		_, err := mgr.RequestUserApprovalAndWait("session-1", nil, 1, 100, 10, "send_message", "test", 1)
		errs <- err
	}()

	waitApprovalPending(t, mgr, "approval-invalid")
	if mgr.ResolveApproval("", "approved") {
		t.Fatal("expected empty approval id to be rejected")
	}
	if mgr.ResolveApproval("approval-invalid", "pending") {
		t.Fatal("expected unsupported status to be rejected")
	}
	if !mgr.ResolveApproval("approval-invalid", "rejected") {
		t.Fatal("expected pending approval to reject successfully")
	}
	if err := <-errs; err != nil {
		t.Fatalf("Request error: %v", err)
	}
}

func TestApprovalManager_RestorePending_BitsUT(t *testing.T) {
	store := &testAgentStore{approvals: map[string]*dal.AgentApproval{
		"approval-restored": {
			ApprovalID: "approval-restored",
			Status:     dal.AgentApprovalStatusPending,
			ExpiresAt:  time.Now().Add(time.Minute).Unix(),
		},
	}}

	mgr := NewApprovalManager(store)
	if !mgr.ResolveApproval("approval-restored", "approved") {
		t.Fatal("expected restored pending approval to resolve")
	}
	approval, err := store.GetApproval("approval-restored")
	if err != nil {
		t.Fatalf("GetApproval error: %v", err)
	}
	if approval.Status != dal.AgentApprovalStatusApproved {
		t.Fatalf("expected restored approval status approved, got %d", approval.Status)
	}
}

func requireApprovalResolved(t *testing.T, mgr *ApprovalManager, approvalID, status string) {
	t.Helper()
	waitApprovalPending(t, mgr, approvalID)
	if !mgr.ResolveApproval(approvalID, status) {
		t.Fatalf("approval %s was not resolved", approvalID)
	}
}

func waitApprovalPending(t *testing.T, mgr *ApprovalManager, approvalID string) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		mgr.mu.Lock()
		_, ok := mgr.pending[approvalID]
		mgr.mu.Unlock()
		if ok {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("approval %s was not pending", approvalID)
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
