package dal

import (
	"encoding/json"

	"gorm.io/gorm"
)

type BotStore interface {
	GetConfig(uid uint64) (*BotConfig, error)
	CreateConfig(config *BotConfig) error
	UpdatePermissions(uid uint64, permissions []string) error
	ListAllConfigs() ([]BotConfig, error)
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
