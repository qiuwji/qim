package agent

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/presence"
	"qim/internal/pkg/pushtype"
)

const defaultApprovalTimeout = 60 * time.Second

type ApprovalManager struct {
	store   dal.AgentStore
	pending map[string]*pendingApproval
	mu      sync.Mutex
	newID   func() string
}

// pendingApproval represents an approval request that is waiting for a user
// decision. resp is nil for approvals restored after restart because the
// original Agent HTTP request no longer exists.
type pendingApproval struct {
	resp  chan approvalResult
	timer *time.Timer
}

type approvalResult struct {
	ID     string
	Status string
}

func NewApprovalManager(store dal.AgentStore) *ApprovalManager {
	m := &ApprovalManager{
		store:   store,
		pending: make(map[string]*pendingApproval),
		newID:   newApprovalID,
	}
	m.restorePending()
	return m
}

func (m *ApprovalManager) Request(
	sessionID string,
	presenceRef *actor.ActorRef,
	ownerUID uint64,
	botUID uint64,
	conversationID uint64,
	action string,
	detail string,
	timeoutSeconds int,
) (map[string]string, error) {
	return m.RequestUserApprovalAndWait(sessionID, presenceRef, ownerUID, botUID, conversationID, action, detail, timeoutSeconds)
}

// RequestUserApprovalAndWait creates one approval request, pushes it to the
// user's online gateways, then waits until the user approves, rejects, or the
// request times out.
func (m *ApprovalManager) RequestUserApprovalAndWait(
	sessionID string,
	presenceRef *actor.ActorRef,
	ownerUID uint64,
	botUID uint64,
	conversationID uint64,
	action string,
	detail string,
	timeoutSeconds int,
) (map[string]string, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = int(defaultApprovalTimeout.Seconds())
	}

	approval, err := m.createPendingApproval(sessionID, ownerUID, botUID, conversationID, action, detail, timeoutSeconds)
	if err != nil {
		return nil, err
	}

	resp := make(chan approvalResult, 1)
	m.registerWaiter(approval.ApprovalID, approval.ExpiresAt, resp)
	pushApprovalRequest(presenceRef, approval, timeoutSeconds)

	return waitApprovalResult(resp), nil
}

func (m *ApprovalManager) createPendingApproval(
	sessionID string,
	ownerUID uint64,
	botUID uint64,
	conversationID uint64,
	action string,
	detail string,
	timeoutSeconds int,
) (*dal.AgentApproval, error) {
	now := time.Now().Unix()
	approval := &dal.AgentApproval{
		ApprovalID:     m.newID(),
		SessionID:      sessionID,
		BotUID:         botUID,
		OwnerUID:       ownerUID,
		ConversationID: conversationID,
		Action:         action,
		Detail:         detail,
		Status:         dal.AgentApprovalStatusPending,
		ExpiresAt:      now + int64(timeoutSeconds),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if m.store == nil {
		return approval, nil
	}
	if err := m.store.CreateApproval(approval); err != nil {
		return nil, err
	}
	return approval, nil
}

func waitApprovalResult(resp <-chan approvalResult) map[string]string {
	result := <-resp
	return map[string]string{"status": result.Status, "approval_id": result.ID}
}

func (m *ApprovalManager) ResolveApproval(approvalID, status string) bool {
	if approvalID == "" || (status != "approved" && status != "rejected") {
		return false
	}
	return m.finishApproval(approvalID, statusToApprovalState(status))
}

func (m *ApprovalManager) restorePending() {
	if m.store == nil {
		return
	}
	approvals, err := m.store.ListPendingApprovals()
	if err != nil {
		return
	}
	now := time.Now().Unix()
	for _, approval := range approvals {
		if approval.ExpiresAt <= now {
			_, _ = m.store.ResolveApproval(approval.ApprovalID, dal.AgentApprovalStatusTimeout, now)
			continue
		}
		m.registerWaiter(approval.ApprovalID, approval.ExpiresAt, nil)
	}
}

func (m *ApprovalManager) registerWaiter(approvalID string, expiresAt int64, resp chan approvalResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.pending[approvalID]; ok {
		if current.timer != nil {
			current.timer.Stop()
		}
	}
	ttl := time.Until(time.Unix(expiresAt, 0))
	if ttl < 0 {
		ttl = 0
	}
	m.pending[approvalID] = &pendingApproval{
		resp: resp,
		timer: time.AfterFunc(ttl, func() {
			m.finishApproval(approvalID, dal.AgentApprovalStatusTimeout)
		}),
	}
}

func (m *ApprovalManager) finishApproval(approvalID string, status int8) bool {
	resultStatus := approvalStateToStatus(status)
	if m.store != nil {
		if _, err := m.store.ResolveApproval(approvalID, status, time.Now().Unix()); err != nil {
			return false
		}
	}
	m.mu.Lock()
	entry, ok := m.pending[approvalID]
	if ok {
		delete(m.pending, approvalID)
	}
	m.mu.Unlock()
	if !ok {
		return m.store != nil
	}
	if entry.timer != nil {
		entry.timer.Stop()
	}
	if entry.resp != nil {
		entry.resp <- approvalResult{ID: approvalID, Status: resultStatus}
	}
	return true
}

func pushApprovalRequest(
	presenceRef *actor.ActorRef,
	approval *dal.AgentApproval,
	timeoutSeconds int,
) {
	if presenceRef == nil || approval == nil {
		return
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: approval.OwnerUID}, askTimeout)
	if err != nil {
		return
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok {
		return
	}
	data := map[string]any{
		"approval_id":     approval.ApprovalID,
		"bot_uid":         approval.BotUID,
		"conversation_id": approval.ConversationID,
		"action":          approval.Action,
		"detail":          approval.Detail,
		"timeout_seconds": timeoutSeconds,
	}
	for _, gw := range result.Gateways {
		if gw != nil {
			_ = gw.Tell(pushtype.PushCmd{Type: "agent", Action: "approval_request", Data: data})
		}
	}
}

func newApprovalID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}

func statusToApprovalState(status string) int8 {
	switch status {
	case "approved":
		return dal.AgentApprovalStatusApproved
	case "rejected":
		return dal.AgentApprovalStatusRejected
	default:
		return dal.AgentApprovalStatusTimeout
	}
}

func approvalStateToStatus(status int8) string {
	switch status {
	case dal.AgentApprovalStatusApproved:
		return "approved"
	case dal.AgentApprovalStatusRejected:
		return "rejected"
	default:
		return "timeout"
	}
}
