package dal

import (
	"errors"
	"time"

	"qim/internal/domain/conversation/store"

	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormConvStore struct {
	db *gorm.DB
}

func NewConvStore(db *gorm.DB) store.Store {
	return &gormConvStore{db: db}
}

func (s *gormConvStore) GetConversation(id uint64) (*store.ConversationRecord, error) {
	var conv Conversation
	if err := s.db.Where("id = ? AND status = 0", id).First(&conv).Error; err != nil {
		return nil, err
	}
	return toConversationRecord(conv), nil
}

func (s *gormConvStore) CreateConversation(conv *store.ConversationRecord) error {
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

func (s *gormConvStore) CreatePrivateConversation(input store.CreatePrivateConversationInput) (*store.ConversationRecord, error) {
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
		Type:        int8(store.ConvTypePrivate),
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
			{ConversationID: conv.ID, UserID: input.UID1, Role: int8(store.MemberRoleRegular), JoinTime: now},
			{ConversationID: conv.ID, UserID: input.UID2, Role: int8(store.MemberRoleRegular), JoinTime: now},
		}
		if err := tx.Create(&members).Error; err != nil {
			return err
		}
		ucs := []UserConversation{
			{UserID: input.UID1, ConversationID: conv.ID, LastMsgAt: now},
			{UserID: input.UID2, ConversationID: conv.ID, LastMsgAt: now},
		}
		return tx.Create(&ucs).Error
	}); err != nil {
		return nil, err
	}
	return toConversationRecord(*conv), nil
}

func (s *gormConvStore) CreateGroupConversation(input store.CreateGroupConversationInput) (*store.ConversationRecord, error) {
	now := input.CreatedAt
	if now == 0 {
		now = time.Now().Unix()
	}
	conv := &Conversation{
		Type:        int8(store.ConvTypeGroup),
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
		if err := tx.Create(&members).Error; err != nil {
			return err
		}
		allUIDs := append([]uint64{input.OwnerID}, input.MemberUIDs...)
		ucs := make([]UserConversation, 0, len(allUIDs))
		seen := map[uint64]struct{}{}
		for _, uid := range allUIDs {
			if uid == 0 {
				continue
			}
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
			ucs = append(ucs, UserConversation{UserID: uid, ConversationID: conv.ID, LastMsgAt: now})
		}
		return tx.Create(&ucs).Error
	}); err != nil {
		return nil, err
	}
	return toConversationRecord(*conv), nil
}

func (s *gormConvStore) UpdateConversation(id uint64, updates map[string]any) error {
	return s.db.Model(&Conversation{}).Where("id = ?", id).Updates(updates).Error
}

func (s *gormConvStore) DissolveConversation(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Member{}).Where("conversation_id = ?", id).Update("status", int8(1)).Error; err != nil {
			return err
		}
		return tx.Model(&Conversation{}).Where("id = ?", id).Update("status", int8(1)).Error
	})
}

func (s *gormConvStore) FindPrivateConversation(uid1, uid2 uint64) (*store.ConversationRecord, error) {
	var conv Conversation
	err := s.db.Joins("JOIN members m1 ON m1.conversation_id = conversations.id AND m1.user_id = ? AND m1.status = 0", uid1).
		Joins("JOIN members m2 ON m2.conversation_id = conversations.id AND m2.user_id = ? AND m2.status = 0", uid2).
		Where("conversations.type = ? AND conversations.status = 0", 1).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return toConversationRecord(conv), nil
}

func (s *gormConvStore) GetMembers(convID uint64) ([]store.MemberRecord, error) {
	var members []Member
	if err := s.db.Where("conversation_id = ? AND status = 0", convID).Find(&members).Error; err != nil {
		return nil, err
	}
	records := make([]store.MemberRecord, 0, len(members))
	for _, m := range members {
		records = append(records, toMemberRecord(m))
	}
	return records, nil
}

func (s *gormConvStore) CreateMember(member *store.MemberRecord) error {
	model := Member{
		ConversationID: member.ConversationID,
		UserID:         member.UserID,
		Role:           member.Role,
		LastReadSeq:    member.LastReadSeq,
		JoinTime:       member.JoinTime,
	}
	return s.db.Create(&model).Error
}

func (s *gormConvStore) CreateMembers(members []store.MemberRecord) error {
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
	return s.db.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, uid).Update("status", int8(1)).Error
}

func (s *gormConvStore) UpdateMember(convID, uid uint64, updates map[string]any) error {
	return s.db.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, uid).Updates(updates).Error
}

func (s *gormConvStore) TransferOwner(convID, oldOwnerUID, newOwnerUID uint64) error {
	now := time.Now().Unix()
	tx := s.db.Begin()
	if err := tx.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, oldOwnerUID).
		Update("role", int8(store.MemberRoleRegular)).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&Member{}).Where("conversation_id = ? AND user_id = ?", convID, newOwnerUID).
		Update("role", int8(store.MemberRoleOwner)).Error; err != nil {
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

func (s *gormConvStore) GetUserConversations(uid uint64) ([]store.UserConversationRecord, error) {
	var ucs []UserConversation
	if err := s.db.Where("user_id = ? AND is_deleted = ?", uid, false).Find(&ucs).Error; err != nil {
		return nil, err
	}
	records := make([]store.UserConversationRecord, 0, len(ucs))
	for _, uc := range ucs {
		records = append(records, store.UserConversationRecord{
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

func (s *gormConvStore) MarkConversationRead(uid, convID uint64, seq int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&UserConversation{}).
			Where("user_id = ? AND conversation_id = ?", uid, convID).
			Update("unread_count", 0).Error; err != nil {
			return err
		}
		if seq <= 0 {
			return nil
		}
		return tx.Model(&Member{}).
			Where("conversation_id = ? AND user_id = ?", convID, uid).
			Update("last_read_seq", seq).Error
	})
}

func (s *gormConvStore) CommitMessage(input store.MessageCommitInput) (*store.MessageCommitResult, error) {
	if input.Message.ClientID != "" {
		existing, err := s.findMessageByClientID(input.Message.ConversationID, input.Message.SenderID, input.Message.ClientID)
		if err == nil {
			return toMessageCommitResult(existing, true), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

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
		if input.Message.ClientID != "" && isUniqueConstraintError(err) {
			existing, findErr := s.findMessageByClientID(input.Message.ConversationID, input.Message.SenderID, input.Message.ClientID)
			if findErr == nil {
				return toMessageCommitResult(existing, true), nil
			}
		}
		return nil, err
	}

	return toMessageCommitResult(msg, false), nil
}

func (s *gormConvStore) GetMessage(convID, messageID uint64) (*store.MessageRecord, error) {
	var msg Message
	if err := s.db.Where("conversation_id = ? AND id = ?", convID, messageID).First(&msg).Error; err != nil {
		return nil, err
	}
	return &store.MessageRecord{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Seq:            msg.Seq,
		SenderID:       msg.SenderID,
		Revoked:        msg.Revoked,
	}, nil
}

func (s *gormConvStore) RevokeMessage(convID, messageID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var msg Message
		if err := tx.Where("conversation_id = ? AND id = ?", convID, messageID).First(&msg).Error; err != nil {
			return err
		}
		if msg.Revoked {
			return nil
		}
		if err := tx.Model(&Message{}).
			Where("conversation_id = ? AND id = ?", convID, messageID).
			Update("revoked", true).Error; err != nil {
			return err
		}
		return tx.Model(&UserConversation{}).
			Where("conversation_id = ? AND user_id <> ? AND unread_count > 0", convID, msg.SenderID).
			Where("user_id IN (?)", tx.Model(&Member{}).
				Select("user_id").
				Where("conversation_id = ? AND status = 0 AND last_read_seq < ?", convID, msg.Seq)).
			Update("unread_count", gorm.Expr("unread_count - 1")).Error
	})
}

func (s *gormConvStore) findMessageByClientID(convID, senderID uint64, clientID string) (*Message, error) {
	var msg Message
	err := s.db.Where(
		"conversation_id = ? AND sender_id = ? AND client_id = ?",
		convID,
		senderID,
		clientID,
	).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *gormConvStore) appendMessage(tx *gorm.DB, msg *Message) error {
	return tx.Create(msg).Error
}

func toMessageCommitResult(msg *Message, duplicated bool) *store.MessageCommitResult {
	result := &store.MessageCommitResult{
		MessageID:  msg.ID,
		Seq:        msg.Seq,
		SenderID:   msg.SenderID,
		MsgType:    msg.MsgType,
		ReplyTo:    msg.ReplyTo,
		ClientID:   msg.ClientID,
		CreatedAt:  msg.CreatedAt,
		Duplicated: duplicated,
		Revoked:    msg.Revoked,
	}
	if !msg.Revoked {
		result.Content = msg.Content
	}
	return result
}

func isUniqueConstraintError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique ||
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey
	}
	return false
}

func (s *gormConvStore) updateConversationSeq(tx *gorm.DB, input store.MessageAppendInput) error {
	return tx.Model(&Conversation{}).Where("id = ?", input.ConversationID).
		Updates(map[string]any{"max_seq": input.Seq, "updated_at": input.CreatedAt}).Error
}

func (s *gormConvStore) projectUnread(tx *gorm.DB, input store.UnreadProjectionInput) error {
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

func toConversationRecord(conv Conversation) *store.ConversationRecord {
	return &store.ConversationRecord{
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

func toMemberRecord(member Member) store.MemberRecord {
	return store.MemberRecord{
		ConversationID: member.ConversationID,
		UserID:         member.UserID,
		Role:           member.Role,
		LastReadSeq:    member.LastReadSeq,
		JoinTime:       member.JoinTime,
	}
}

func groupMembers(convID, ownerID uint64, memberUIDs []uint64, joinTime int64) []Member {
	members := []Member{
		{ConversationID: convID, UserID: ownerID, Role: int8(store.MemberRoleOwner), JoinTime: joinTime},
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
			Role:           int8(store.MemberRoleRegular),
			JoinTime:       joinTime,
		})
	}
	return members
}
