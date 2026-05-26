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

func newBotTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&BotConfig{}); err != nil {
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
