package dal

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const AgentSessionStatusActive int8 = 0

const (
	AgentApprovalStatusPending int8 = iota
	AgentApprovalStatusApproved
	AgentApprovalStatusRejected
	AgentApprovalStatusTimeout
)

type AgentStore interface {
	CreateSession(session *AgentSession) error
	GetSession(sessionID string) (*AgentSession, error)
	TouchSession(sessionID string, lastSeenAt, expiresAt int64) error

	UpsertSubscription(sub *AgentSubscription) error
	DeleteSubscriptions(sessionID string, botUID uint64, eventNames []string) error
	ListSubscriptions() ([]AgentSubscription, error)

	CreateApproval(approval *AgentApproval) error
	ResolveApproval(approvalID string, status int8, resolvedAt int64) (*AgentApproval, error)
	ListPendingApprovals() ([]AgentApproval, error)
	GetApproval(approvalID string) (*AgentApproval, error)
}

type gormAgentStore struct {
	db *gorm.DB
}

func NewAgentStore(db *gorm.DB) AgentStore {
	return &gormAgentStore{db: db}
}

func (s *gormAgentStore) CreateSession(session *AgentSession) error {
	return s.db.Create(session).Error
}

func (s *gormAgentStore) GetSession(sessionID string) (*AgentSession, error) {
	var session AgentSession
	if err := s.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *gormAgentStore) TouchSession(sessionID string, lastSeenAt, expiresAt int64) error {
	return s.db.Model(&AgentSession{}).
		Where("session_id = ? AND status = ?", sessionID, AgentSessionStatusActive).
		Updates(map[string]any{
			"last_seen_at": lastSeenAt,
			"expires_at":   expiresAt,
			"updated_at":   lastSeenAt,
		}).Error
}

func (s *gormAgentStore) UpsertSubscription(sub *AgentSubscription) error {
	var existing AgentSubscription
	err := s.db.Where(
		"session_id = ? AND bot_uid = ? AND event_name = ? AND filter_json = ?",
		sub.SessionID, sub.BotUID, sub.EventName, sub.FilterJSON,
	).First(&existing).Error
	switch {
	case err == nil:
		return s.db.Model(&AgentSubscription{}).Where("id = ?", existing.ID).
			Update("updated_at", sub.UpdatedAt).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		return s.db.Create(sub).Error
	default:
		return err
	}
}

func (s *gormAgentStore) DeleteSubscriptions(sessionID string, botUID uint64, eventNames []string) error {
	tx := s.db.Where("session_id = ? AND bot_uid = ?", sessionID, botUID)
	if len(eventNames) > 0 {
		tx = tx.Where("event_name IN ?", eventNames)
	}
	return tx.Delete(&AgentSubscription{}).Error
}

func (s *gormAgentStore) ListSubscriptions() ([]AgentSubscription, error) {
	var subs []AgentSubscription
	if err := s.db.Order("id ASC").Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *gormAgentStore) CreateApproval(approval *AgentApproval) error {
	return s.db.Create(approval).Error
}

func (s *gormAgentStore) ResolveApproval(approvalID string, status int8, resolvedAt int64) (*AgentApproval, error) {
	var approval AgentApproval
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("approval_id = ?", approvalID).First(&approval).Error; err != nil {
			return err
		}
		if approval.Status != AgentApprovalStatusPending {
			return gorm.ErrRecordNotFound
		}
		approval.Status = status
		approval.ResolvedAt = resolvedAt
		approval.UpdatedAt = resolvedAt
		return tx.Save(&approval).Error
	})
	if err != nil {
		return nil, err
	}
	return &approval, nil
}

func (s *gormAgentStore) ListPendingApprovals() ([]AgentApproval, error) {
	var approvals []AgentApproval
	if err := s.db.Where("status = ?", AgentApprovalStatusPending).Find(&approvals).Error; err != nil {
		return nil, err
	}
	return approvals, nil
}

func (s *gormAgentStore) GetApproval(approvalID string) (*AgentApproval, error) {
	var approval AgentApproval
	if err := s.db.Where("approval_id = ?", approvalID).First(&approval).Error; err != nil {
		return nil, err
	}
	return &approval, nil
}

func NewAgentSession(sessionID string, ttl time.Duration) *AgentSession {
	now := time.Now().Unix()
	return &AgentSession{
		SessionID:  sessionID,
		Status:     AgentSessionStatusActive,
		LastSeenAt: now,
		ExpiresAt:  now + int64(ttl.Seconds()),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
