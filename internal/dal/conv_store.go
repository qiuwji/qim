package dal

import (
	"time"

	"gorm.io/gorm"
)

type ConvStore interface {
	GetConversation(id uint64) (*Conversation, error)
	CreateConversation(conv *Conversation) error
	UpdateConversation(id uint64, updates map[string]any) error
	DeleteConversation(id uint64) error
	FindPrivateConversation(uid1, uid2 uint64) (*Conversation, error)

	GetMembers(convID uint64) ([]Member, error)
	CreateMember(member *Member) error
	CreateMembers(members []Member) error
	DeleteMember(convID, uid uint64) error
	UpdateMember(convID, uid uint64, updates map[string]any) error
	DeleteAllMembers(convID uint64) error
	TransferOwner(convID, oldOwnerUID, newOwnerUID uint64) error

	GetUserConversations(uid uint64) ([]UserConversation, error)
	UpdateUserConversation(uid, convID uint64, updates map[string]any) error
}

type gormConvStore struct {
	db *gorm.DB
}

func NewConvStore(db *gorm.DB) ConvStore {
	return &gormConvStore{db: db}
}

func (s *gormConvStore) GetConversation(id uint64) (*Conversation, error) {
	var conv Conversation
	if err := s.db.First(&conv, id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (s *gormConvStore) CreateConversation(conv *Conversation) error {
	return s.db.Create(conv).Error
}

func (s *gormConvStore) UpdateConversation(id uint64, updates map[string]any) error {
	return s.db.Model(&Conversation{}).Where("id = ?", id).Updates(updates).Error
}

func (s *gormConvStore) DeleteConversation(id uint64) error {
	return s.db.Delete(&Conversation{}, id).Error
}

func (s *gormConvStore) FindPrivateConversation(uid1, uid2 uint64) (*Conversation, error) {
	var conv Conversation
	err := s.db.Joins("JOIN members m1 ON m1.conversation_id = conversations.id AND m1.user_id = ?", uid1).
		Joins("JOIN members m2 ON m2.conversation_id = conversations.id AND m2.user_id = ?", uid2).
		Where("conversations.type = ?", 1). // ConvTypePrivate
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (s *gormConvStore) GetMembers(convID uint64) ([]Member, error) {
	var members []Member
	if err := s.db.Where("conversation_id = ?", convID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (s *gormConvStore) CreateMember(member *Member) error {
	return s.db.Create(member).Error
}

func (s *gormConvStore) CreateMembers(members []Member) error {
	return s.db.Create(&members).Error
}

func (s *gormConvStore) DeleteMember(convID, uid uint64) error {
	return s.db.Where("conversation_id = ? AND user_id = ?", convID, uid).Delete(&Member{}).Error
}

func (s *gormConvStore) UpdateMember(convID, uid uint64, updates map[string]any) error {
	return s.db.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, uid).Updates(updates).Error
}

func (s *gormConvStore) DeleteAllMembers(convID uint64) error {
	return s.db.Where("conversation_id = ?", convID).Delete(&Member{}).Error
}

func (s *gormConvStore) TransferOwner(convID, oldOwnerUID, newOwnerUID uint64) error {
	now := time.Now().Unix()
	tx := s.db.Begin()
	if err := tx.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, oldOwnerUID).
		Update("role", int8(0)).Error; err != nil { // MemberRoleRegular
		tx.Rollback()
		return err
	}
	if err := tx.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, newOwnerUID).
		Update("role", int8(2)).Error; err != nil { // MemberRoleOwner
		tx.Rollback()
		return err
	}
	if err := tx.Model(&Conversation{}).Where("id = ?", convID).
		Updates(map[string]any{"owner_id": newOwnerUID, "updated_at": now}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *gormConvStore) GetUserConversations(uid uint64) ([]UserConversation, error) {
	var ucs []UserConversation
	if err := s.db.Where("user_id = ? AND is_deleted = ?", uid, false).Find(&ucs).Error; err != nil {
		return nil, err
	}
	return ucs, nil
}

func (s *gormConvStore) UpdateUserConversation(uid, convID uint64, updates map[string]any) error {
	return s.db.Model(&UserConversation{}).Where("user_id = ? AND conversation_id = ?", uid, convID).Updates(updates).Error
}
