package dal

import (
	"errors"
	"time"

	convdomain "qim/internal/domain/conversation"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormConvStore struct {
	db *gorm.DB
}

func NewConvStore(db *gorm.DB) convdomain.Store {
	return &gormConvStore{db: db}
}

func (s *gormConvStore) GetConversation(id uint64) (*convdomain.ConversationRecord, error) {
	var conv Conversation
	if err := s.db.First(&conv, id).Error; err != nil {
		return nil, err
	}
	return toConversationRecord(conv), nil
}

func (s *gormConvStore) CreateConversation(conv *convdomain.ConversationRecord) error {
	model := Conversation{
		Type:        conv.Type,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		OwnerID:     conv.OwnerID,
		MaxSeq:      conv.MaxSeq,
		MemberLimit: conv.MemberLimit,
		CreatedAt:   conv.CreatedAt,
		UpdatedAt:   conv.UpdatedAt,
	}
	if err := s.db.Create(&model).Error; err != nil {
		return err
	}
	conv.ID = model.ID
	return nil
}

func (s *gormConvStore) CreatePrivateConversation(input convdomain.CreatePrivateConversationInput) (*convdomain.ConversationRecord, error) {
	existing, err := s.FindPrivateConversation(input.UID1, input.UID2)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := input.CreatedAt
	if now == 0 {
		now = time.Now().Unix()
	}
	conv := &Conversation{
		Type:        int8(convdomain.ConvTypePrivate),
		OwnerID:     input.UID1,
		MemberLimit: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conv).Error; err != nil {
			return err
		}
		members := []Member{
			{ConversationID: conv.ID, UserID: input.UID1, Role: int8(convdomain.MemberRoleRegular), JoinTime: now},
			{ConversationID: conv.ID, UserID: input.UID2, Role: int8(convdomain.MemberRoleRegular), JoinTime: now},
		}
		return tx.Create(&members).Error
	}); err != nil {
		return nil, err
	}
	return toConversationRecord(*conv), nil
}

func (s *gormConvStore) CreateGroupConversation(input convdomain.CreateGroupConversationInput) (*convdomain.ConversationRecord, error) {
	now := input.CreatedAt
	if now == 0 {
		now = time.Now().Unix()
	}
	conv := &Conversation{
		Type:        int8(convdomain.ConvTypeGroup),
		Name:        input.Name,
		Avatar:      input.Avatar,
		OwnerID:     input.OwnerID,
		MemberLimit: 500,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conv).Error; err != nil {
			return err
		}
		members := groupMembers(conv.ID, input.OwnerID, input.MemberUIDs, now)
		return tx.Create(&members).Error
	}); err != nil {
		return nil, err
	}
	return toConversationRecord(*conv), nil
}

func (s *gormConvStore) UpdateConversation(id uint64, updates map[string]any) error {
	return s.db.Model(&Conversation{}).Where("id = ?", id).Updates(updates).Error
}

func (s *gormConvStore) DeleteConversation(id uint64) error {
	return s.db.Delete(&Conversation{}, id).Error
}

func (s *gormConvStore) FindPrivateConversation(uid1, uid2 uint64) (*convdomain.ConversationRecord, error) {
	var conv Conversation
	err := s.db.Joins("JOIN members m1 ON m1.conversation_id = conversations.id AND m1.user_id = ?", uid1).
		Joins("JOIN members m2 ON m2.conversation_id = conversations.id AND m2.user_id = ?", uid2).
		Where("conversations.type = ?", 1). // ConvTypePrivate
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return toConversationRecord(conv), nil
}

func (s *gormConvStore) GetMembers(convID uint64) ([]convdomain.MemberRecord, error) {
	var members []Member
	if err := s.db.Where("conversation_id = ?", convID).Find(&members).Error; err != nil {
		return nil, err
	}
	records := make([]convdomain.MemberRecord, 0, len(members))
	for _, m := range members {
		records = append(records, toMemberRecord(m))
	}
	return records, nil
}

func (s *gormConvStore) CreateMember(member *convdomain.MemberRecord) error {
	model := Member{
		ConversationID: member.ConversationID,
		UserID:         member.UserID,
		Role:           member.Role,
		LastReadSeq:    member.LastReadSeq,
		JoinTime:       member.JoinTime,
	}
	return s.db.Create(&model).Error
}

func (s *gormConvStore) CreateMembers(members []convdomain.MemberRecord) error {
	models := make([]Member, 0, len(members))
	for _, m := range members {
		models = append(models, Member{
			ConversationID: m.ConversationID,
			UserID:         m.UserID,
			Role:           m.Role,
			LastReadSeq:    m.LastReadSeq,
			JoinTime:       m.JoinTime,
		})
	}
	return s.db.Create(&models).Error
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

func (s *gormConvStore) GetUserConversations(uid uint64) ([]convdomain.UserConversationRecord, error) {
	var ucs []UserConversation
	if err := s.db.Where("user_id = ? AND is_deleted = ?", uid, false).Find(&ucs).Error; err != nil {
		return nil, err
	}
	records := make([]convdomain.UserConversationRecord, 0, len(ucs))
	for _, uc := range ucs {
		records = append(records, convdomain.UserConversationRecord{
			ConversationID: uc.ConversationID,
			IsPinned:       uc.IsPinned,
			IsMuted:        uc.IsMuted,
			UnreadCount:    uc.UnreadCount,
			LastMsgAt:      uc.LastMsgAt,
		})
	}
	return records, nil
}

func (s *gormConvStore) UpdateUserConversation(uid, convID uint64, updates map[string]any) error {
	return s.db.Model(&UserConversation{}).Where("user_id = ? AND conversation_id = ?", uid, convID).Updates(updates).Error
}

func (s *gormConvStore) CommitMessage(input convdomain.MessageCommitInput) (*convdomain.MessageCommitResult, error) {
	msg := &Message{
		ConversationID: input.Message.ConversationID,
		Seq:            input.Message.Seq,
		SenderID:       input.Message.SenderID,
		MsgType:        input.Message.MsgType,
		Content:        input.Message.Content,
		ReplyTo:        input.Message.ReplyTo,
		ClientID:       input.Message.ClientID,
		CreatedAt:      input.Message.CreatedAt,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.appendMessage(tx, msg); err != nil {
			return err
		}
		if err := s.updateConversationSeq(tx, input.Message); err != nil {
			return err
		}
		if err := s.projectUnread(tx, input.UnreadProjection); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &convdomain.MessageCommitResult{MessageID: msg.ID}, nil
}

func (s *gormConvStore) appendMessage(tx *gorm.DB, msg *Message) error {
	return tx.Create(msg).Error
}

func (s *gormConvStore) updateConversationSeq(tx *gorm.DB, input convdomain.MessageAppendInput) error {
	return tx.Model(&Conversation{}).Where("id = ?", input.ConversationID).
		Updates(map[string]any{"max_seq": input.Seq, "updated_at": input.CreatedAt}).Error
}

func (s *gormConvStore) projectUnread(tx *gorm.DB, input convdomain.UnreadProjectionInput) error {
	for _, uid := range input.MemberUIDs {
		unreadDelta := 1
		if uid == input.SenderID {
			unreadDelta = 0
		}
		uc := UserConversation{
			UserID:         uid,
			ConversationID: input.ConversationID,
			LastMsgAt:      input.LastMsgAt,
			UnreadCount:    unreadDelta,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "conversation_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"is_deleted":   false,
				"last_msg_at":  input.LastMsgAt,
				"unread_count": gorm.Expr("unread_count + ?", unreadDelta),
			}),
		}).Create(&uc).Error; err != nil {
			return err
		}
	}
	return nil
}

func toConversationRecord(conv Conversation) *convdomain.ConversationRecord {
	return &convdomain.ConversationRecord{
		ID:          conv.ID,
		Type:        conv.Type,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		OwnerID:     conv.OwnerID,
		MaxSeq:      conv.MaxSeq,
		MemberLimit: conv.MemberLimit,
		CreatedAt:   conv.CreatedAt,
		UpdatedAt:   conv.UpdatedAt,
	}
}

func toMemberRecord(member Member) convdomain.MemberRecord {
	return convdomain.MemberRecord{
		ConversationID: member.ConversationID,
		UserID:         member.UserID,
		Role:           member.Role,
		LastReadSeq:    member.LastReadSeq,
		JoinTime:       member.JoinTime,
	}
}

func groupMembers(convID, ownerID uint64, memberUIDs []uint64, joinTime int64) []Member {
	members := []Member{
		{ConversationID: convID, UserID: ownerID, Role: int8(convdomain.MemberRoleOwner), JoinTime: joinTime},
	}
	seen := map[uint64]struct{}{ownerID: {}}
	for _, uid := range memberUIDs {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		members = append(members, Member{
			ConversationID: convID,
			UserID:         uid,
			Role:           int8(convdomain.MemberRoleRegular),
			JoinTime:       joinTime,
		})
	}
	return members
}
