package dal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestBotStore_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewBotStore(db)

	cfg := &BotConfig{
		UID:       100,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := store.CreateConfig(cfg); err != nil {
		t.Fatalf("CreateConfig error: %v", err)
	}
	if cfg.ID == 0 {
		t.Fatal("expected auto id")
	}

	got, err := store.GetConfig(100)
	if err != nil {
		t.Fatalf("GetConfig error: %v", err)
	}
	if got.UID != 100 {
		t.Fatalf("got UID = %d, want 100", got.UID)
	}

	perms := []string{"friend:read", "message:search"}
	if err := store.UpdatePermissions(100, perms); err != nil {
		t.Fatalf("UpdatePermissions error: %v", err)
	}

	got2, err := store.GetConfig(100)
	if err != nil {
		t.Fatalf("GetConfig after update error: %v", err)
	}
	var decoded []string
	if err := json.Unmarshal([]byte(got2.Permissions), &decoded); err != nil {
		t.Fatalf("unmarshal permissions: %v", err)
	}
	if len(decoded) != 2 || decoded[0] != "friend:read" || decoded[1] != "message:search" {
		t.Fatalf("unexpected permissions: %v", decoded)
	}

	cfg2 := &BotConfig{
		UID:       200,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := store.CreateConfig(cfg2); err != nil {
		t.Fatalf("CreateConfig error: %v", err)
	}

	all, err := store.ListAllConfigs()
	if err != nil {
		t.Fatalf("ListAllConfigs error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ListAllConfigs len = %d, want 2", len(all))
	}
}

func TestBotStoreNotFound_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewBotStore(db)

	_, err := store.GetConfig(999)
	if err == nil {
		t.Fatal("expected error for nonexistent config")
	}
}

func TestBotStoreEmptyPermissions_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewBotStore(db)

	cfg := &BotConfig{
		UID:       300,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := store.CreateConfig(cfg); err != nil {
		t.Fatalf("CreateConfig error: %v", err)
	}

	got, err := store.GetConfig(300)
	if err != nil {
		t.Fatalf("GetConfig error: %v", err)
	}
	if got.Permissions != "" && got.Permissions != "[]" {
		t.Fatalf("expected empty permissions, got %s", got.Permissions)
	}
}

func TestBotStoreActivateBotTx_BitsUT(t *testing.T) {
	db := newBotTestDB(t)
	store := NewBotStore(db)
	creator := User{
		Username:     "owner",
		Password:     "",
		Nickname:     "Owner",
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		LastOnlineAt: time.Now().Unix(),
	}
	if err := db.Create(&creator).Error; err != nil {
		t.Fatalf("create creator: %v", err)
	}

	first, err := store.ActivateBotTx(creator.ID)
	if err != nil {
		t.Fatalf("ActivateBotTx first error: %v", err)
	}
	if first.BotUser.ID == 0 || first.BotUser.UserType != 1 || first.BotUser.CreatorUID != creator.ID {
		t.Fatalf("unexpected bot user: %+v", first.BotUser)
	}
	if first.ConversationID == 0 {
		t.Fatal("expected conversation id")
	}

	second, err := store.ActivateBotTx(creator.ID)
	if err != nil {
		t.Fatalf("ActivateBotTx second error: %v", err)
	}
	if second.BotUser.ID != first.BotUser.ID || second.ConversationID != first.ConversationID {
		t.Fatalf("activation not idempotent: first=%+v second=%+v", first, second)
	}

	var botCount int64
	if err := db.Model(&User{}).Where("creator_uid = ? AND user_type = ?", creator.ID, int8(1)).Count(&botCount).Error; err != nil {
		t.Fatalf("count bots: %v", err)
	}
	if botCount != 1 {
		t.Fatalf("bot count = %d, want 1", botCount)
	}
	var friendCount int64
	if err := db.Model(&Friend{}).Where("user_id IN ? AND friend_uid IN ?", []uint64{creator.ID, first.BotUser.ID}, []uint64{creator.ID, first.BotUser.ID}).Count(&friendCount).Error; err != nil {
		t.Fatalf("count friends: %v", err)
	}
	if friendCount != 2 {
		t.Fatalf("friend edges = %d, want 2", friendCount)
	}
}

func newBotTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&BotConfig{},
		&User{},
		&Friend{},
		&Conversation{},
		&Member{},
		&UserConversation{},
		&AgentSession{},
		&AgentSubscription{},
		&AgentApproval{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
	return db
}
