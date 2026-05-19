package dal

import (
	"fmt"

	"qim/internal/domain/call"

	"gorm.io/gorm"
)

type gormCallStore struct {
	db *gorm.DB
}

func NewCallStore(db *gorm.DB) call.CallStore {
	return &gormCallStore{db: db}
}

func (s *gormCallStore) Create(c *call.Call) error {
	record := toCallRecord(c)
	if err := s.db.Create(record).Error; err != nil {
		return err
	}
	c.ID = record.ID
	return nil
}

func (s *gormCallStore) GetByID(id uint64) (*call.Call, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *gormCallStore) ListByUser(uid uint64, offset, limit int) ([]call.Call, error) {
	return nil, fmt.Errorf("not implemented")
}

func toCallRecord(c *call.Call) *CallRecord {
	return &CallRecord{
		ID:        c.ID,
		CallerUID: c.CallerUID,
		CalleeUID: c.CalleeUID,
		CallType:  c.CallType,
		Status:    int8(c.Status),
		StartedAt: c.StartedAt,
		EndedAt:   c.EndedAt,
		Duration:  c.Duration,
		EndReason: c.EndReason,
		CreatedAt: c.CreatedAt,
	}
}
