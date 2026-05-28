package dal

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	convstore "qim/internal/domain/conversation/store"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BotStore interface {
	GetConfig(uid uint64) (*BotConfig, error)
	CreateConfig(config *BotConfig) error
	UpdatePermissions(uid uint64, permissions []string) error
	ListAllConfigs() ([]BotConfig, error)
	ActivateBotTx(creatorUID uint64) (*BotActivationResult, error)
}

type BotActivationResult struct {
	BotUser        User
	Config         BotConfig
	ConversationID uint64
}

type gormBotStore struct {
	db *gorm.DB
}

func NewBotStore(db *gorm.DB) BotStore {
	return &gormBotStore{db: db}
}

func (s *gormBotStore) GetConfig(uid uint64) (*BotConfig, error) {
	var cfg BotConfig
	if err := s.db.Where("uid = ?", uid).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *gormBotStore) CreateConfig(config *BotConfig) error {
	return s.db.Create(config).Error
}

func (s *gormBotStore) UpdatePermissions(uid uint64, permissions []string) error {
	data, err := json.Marshal(permissions)
	if err != nil {
		return err
	}
	return s.db.Model(&BotConfig{}).Where("uid = ?", uid).Update("permissions", string(data)).Error
}

func (s *gormBotStore) ListAllConfigs() ([]BotConfig, error) {
	var configs []BotConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *gormBotStore) ActivateBotTx(creatorUID uint64) (*BotActivationResult, error) {
	var result BotActivationResult
	now := time.Now().Unix()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		bot, err := findBotUserTx(tx, creatorUID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			bot = &User{
				Username:     fmt.Sprintf("bot_%d", creatorUID),
				Password:     "",
				Nickname:     "AI Bot",
				Avatar:       "",
				UserType:     1,
				CreatorUID:   creatorUID,
				Status:       0,
				CreatedAt:    now,
				UpdatedAt:    now,
				LastOnlineAt: now,
			}
			if err := tx.Create(bot).Error; err != nil {
				return err
			}
		}

		cfg, err := upsertBotConfigTx(tx, bot.ID, now)
		if err != nil {
			return err
		}
		if err := upsertFriendEdges(tx, creatorUID, bot.ID, now).Error; err != nil {
			return err
		}
		convID, err := upsertPrivateConversationTx(tx, creatorUID, bot.ID, now)
		if err != nil {
			return err
		}
		result = BotActivationResult{BotUser: *bot, Config: *cfg, ConversationID: convID}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func findBotUserTx(tx *gorm.DB, creatorUID uint64) (*User, error) {
	var bot User
	if err := tx.Where("creator_uid = ? AND user_type = ?", creatorUID, int8(1)).First(&bot).Error; err != nil {
		return nil, err
	}
	return &bot, nil
}

func upsertBotConfigTx(tx *gorm.DB, botUID uint64, now int64) (*BotConfig, error) {
	cfg := BotConfig{UID: botUID, CreatedAt: now, UpdatedAt: now}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uid"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(&cfg).Error; err != nil {
		return nil, err
	}
	var got BotConfig
	if err := tx.Where("uid = ?", botUID).First(&got).Error; err != nil {
		return nil, err
	}
	return &got, nil
}

func upsertPrivateConversationTx(tx *gorm.DB, uid1, uid2 uint64, now int64) (uint64, error) {
	var existing Conversation
	err := tx.Joins("JOIN members m1 ON m1.conversation_id = conversations.id AND m1.user_id = ? AND m1.status = 0", uid1).
		Joins("JOIN members m2 ON m2.conversation_id = conversations.id AND m2.user_id = ? AND m2.status = 0", uid2).
		Where("conversations.type = ? AND conversations.status = 0", int8(convstore.ConvTypePrivate)).
		First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	conv := Conversation{
		Type:        int8(convstore.ConvTypePrivate),
		OwnerID:     uid1,
		MemberLimit: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tx.Create(&conv).Error; err != nil {
		return 0, err
	}
	members := []Member{
		{ConversationID: conv.ID, UserID: uid1, Role: int8(convstore.MemberRoleRegular), JoinTime: now},
		{ConversationID: conv.ID, UserID: uid2, Role: int8(convstore.MemberRoleRegular), JoinTime: now},
	}
	if err := tx.Create(&members).Error; err != nil {
		return 0, err
	}
	ucs := []UserConversation{
		{UserID: uid1, ConversationID: conv.ID, LastMsgAt: now},
		{UserID: uid2, ConversationID: conv.ID, LastMsgAt: now},
	}
	if err := tx.Create(&ucs).Error; err != nil {
		return 0, err
	}
	return conv.ID, nil
}
