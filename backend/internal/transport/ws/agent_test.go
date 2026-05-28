package ws

import (
	"testing"
)

type fakeApprovalResolver struct {
	id     string
	status string
	ok     bool
}

func (r *fakeApprovalResolver) ResolveApproval(id, status string) bool {
	r.id = id
	r.status = status
	return r.ok
}

func TestAgentRouterDispatch_BitsUT(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		data       string
		resolved   bool
		wantStatus string
		wantOK     bool
	}{
		{"approve success", "approve", `{"approval_id":"approval-1"}`, true, "approved", true},
		{"reject success", "reject", `{"approval_id":"approval-2"}`, true, "rejected", true},
		{"approval not found", "approve", `{"approval_id":"missing"}`, false, "approved", false},
		{"missing approval id", "approve", `{}`, true, "", false},
		{"invalid json", "approve", `{bad`, true, "", false},
		{"unknown action", "unknown", `{"approval_id":"approval-1"}`, true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &fakeApprovalResolver{ok: tt.resolved}
			dispatcher := NewDispatcher(nil, nil, nil, nil, nil, nil, nil, resolver)

			resp := dispatcher.Dispatch(1, req("agent", tt.action, tt.data))
			if tt.wantOK && resp.Type == "error" {
				t.Fatalf("expected success, got error: %+v", resp.Error)
			}
			if !tt.wantOK && resp.Type != "error" {
				t.Fatalf("expected error, got %+v", resp)
			}
			if tt.wantStatus != "" && resolver.status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", resolver.status, tt.wantStatus)
			}
		})
	}
}

func TestAgentRouterUnavailable_BitsUT(t *testing.T) {
	dispatcher := NewDispatcher(nil, nil, nil, nil, nil, nil, nil)

	resp := dispatcher.Dispatch(1, req("agent", "approve", `{"approval_id":"approval-1"}`))
	if resp.Type != "error" {
		t.Fatalf("expected error when approval resolver is unavailable, got %+v", resp)
	}
}
