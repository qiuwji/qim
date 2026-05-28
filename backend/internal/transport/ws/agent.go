package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/pkg/apperr"
)

type AgentApprovalResolver interface {
	ResolveApproval(approvalID, status string) bool
}

type agentRouter struct {
	approvals AgentApprovalResolver
}

func (r *agentRouter) dispatch(_ uint64, action string, data json.RawMessage) WsResponse {
	cmd, err := r.resolveAction(action, data)
	if err != nil {
		return errReply(action, err)
	}
	if r.approvals == nil {
		return errReply(action, apperr.New(apperr.CodeInvalidRequest, "agent approval unavailable"))
	}
	if !r.approvals.ResolveApproval(cmd.ApprovalID, cmd.Status) {
		return errReply(action, apperr.New(apperr.CodeBadRequest, "approval not found"))
	}
	return WsResponse{
		Type:   "ack",
		Action: action,
		Data:   map[string]string{"approval_id": cmd.ApprovalID, "status": cmd.Status},
	}
}

type agentApprovalCmd struct {
	ApprovalID string
	Status     string
}

func (r *agentRouter) resolveAction(action string, data json.RawMessage) (agentApprovalCmd, error) {
	var req struct {
		ApprovalID string `json:"approval_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return agentApprovalCmd{}, err
	}
	if req.ApprovalID == "" {
		return agentApprovalCmd{}, fmt.Errorf("approval_id is required")
	}
	switch action {
	case "approve":
		return agentApprovalCmd{ApprovalID: req.ApprovalID, Status: "approved"}, nil
	case "reject":
		return agentApprovalCmd{ApprovalID: req.ApprovalID, Status: "rejected"}, nil
	default:
		return agentApprovalCmd{}, fmt.Errorf("unknown agent action: %s", action)
	}
}
