package dal

import "gorm.io/gorm"

type MsgStore interface {
	CreateMessage(msg *Message) error
	ListMessages(convID uint64, beforeSeq int64, limit int) ([]Message, error)
	SearchMessages(convID uint64, keyword string, limit int) ([]Message, error)
}

type gormMsgStore struct {
	db *gorm.DB
}

func NewMsgStore(db *gorm.DB) MsgStore {
	return &gormMsgStore{db: db}
}

func (s *gormMsgStore) CreateMessage(msg *Message) error {
	return s.db.Create(msg).Error
}

func (s *gormMsgStore) ListMessages(convID uint64, beforeSeq int64, limit int) ([]Message, error) {
	var messages []Message
	q := s.db.Where("conversation_id = ?", convID)
	if beforeSeq > 0 {
		q = q.Where("seq < ?", beforeSeq)
	}
	if limit <= 0 {
		limit = 50
	}
	if err := q.Order("seq DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *gormMsgStore) SearchMessages(convID uint64, keyword string, limit int) ([]Message, error) {
	var messages []Message
	if limit <= 0 {
		limit = 20
	}
	if err := s.db.Where("conversation_id = ? AND content LIKE ? AND revoked = false", convID, "%"+keyword+"%").
		Order("seq DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}
