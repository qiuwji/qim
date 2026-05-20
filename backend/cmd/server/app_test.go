package main

import (
	"os"
	"path/filepath"
	"testing"

	"qim/internal/actor"
	"qim/internal/transport/ws"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitJWT_BitsUT(t *testing.T) {
	os.Setenv("QIM_JWT_SECRET", "test-secret")
	defer os.Unsetenv("QIM_JWT_SECRET")

	jwt := initJWT()
	if jwt == nil {
		t.Fatal("jwt manager is nil")
	}
}

func TestInitJWT_DefaultSecret_BitsUT(t *testing.T) {
	os.Unsetenv("QIM_JWT_SECRET")
	jwt := initJWT()
	if jwt == nil {
		t.Fatal("jwt manager is nil with default secret")
	}
}

func TestInitEngine_BitsUT(t *testing.T) {
	engine := initEngine()
	if engine == nil {
		t.Fatal("engine is nil")
	}
	engine.Shutdown()
}

func TestInitEventBus_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	bus := initEventBus(engine)
	if bus == nil {
		t.Fatal("event bus is nil")
	}
}

func TestInitDB_BitsUT(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	os.Setenv("QIM_DB_PATH", dbPath)
	defer os.Unsetenv("QIM_DB_PATH")

	db := initDB()
	if db == nil {
		t.Fatal("db is nil")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").Scan(&tables)
	if len(tables) < 8 {
		t.Fatalf("expected at least 8 tables, got %d: %v", len(tables), tables)
	}
}

func TestMustCreateMessageClientIndex_AlreadyExists_BitsUT(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test2.db")
	os.Setenv("QIM_DB_PATH", dbPath)
	defer os.Unsetenv("QIM_DB_PATH")

	db := initDB()
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	mustCreateMessageClientIndex(db)
}

func TestInitStores_BitsUT(t *testing.T) {
	db := newMemoryDB(t)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	s := initStores(db)
	if s.conv == nil || s.user == nil || s.msg == nil || s.friend == nil || s.call == nil {
		t.Fatal("one or more stores are nil")
	}
}

func TestInitServices_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	db := newMemoryDB(t)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	s := initStores(db)
	events := initEventBus(engine)

	svcs := initServices(engine, s, events)
	if svcs.conv == nil || svcs.user == nil || svcs.msg == nil || svcs.friend == nil || svcs.call == nil {
		t.Fatal("one or more services are nil")
	}
}

func TestInitActors_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	db := newMemoryDB(t)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	s := initStores(db)
	events := initEventBus(engine)

	initActors(engine, s, events)

	actors := []string{"conv-manager", "user-manager", "msg-reader-0", "msg-reader-1", "msg-reader-2", "msg-reader-3", "friend-manager", "presence", "call-manager"}
	for _, name := range actors {
		if _, ok := engine.Lookup(name); !ok {
			t.Fatalf("actor %s not found", name)
		}
	}
}

func TestInitEventHandlers_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	db := newMemoryDB(t)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	s := initStores(db)
	events := initEventBus(engine)
	initActors(engine, s, events)

	initEventHandlers(engine, events)

	if _, ok := engine.Lookup("message-push"); !ok {
		t.Fatal("message-push actor not found")
	}
}

func TestInitHandlers_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	db := newMemoryDB(t)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	s := initStores(db)
	events := initEventBus(engine)
	initActors(engine, s, events)
	svcs := initServices(engine, s, events)
	jwt := initJWT()

	handlers := initHandlers(svcs, jwt)
	if handlers.Conv == nil || handlers.User == nil || handlers.Msg == nil || handlers.Friend == nil || handlers.File == nil {
		t.Fatal("one or more handlers are nil")
	}
}

func TestInitDispatcher_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	db := newMemoryDB(t)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	s := initStores(db)
	events := initEventBus(engine)
	initActors(engine, s, events)
	svcs := initServices(engine, s, events)

	dispatcher := initDispatcher(svcs, engine)
	if dispatcher == nil {
		t.Fatal("dispatcher is nil")
	}
	resp := dispatcher.Dispatch(0, ws.WsRequest{Type: "conv", Action: "list", Data: nil})
	if resp.Type == "" {
		t.Fatal("dispatcher returned empty response")
	}
}

func TestMustSpawn_BitsUT(t *testing.T) {
	engine := initEngine()
	defer engine.Shutdown()

	ref := mustSpawn(engine, "test-actor", actorFunc(func(ctx actor.Context) {}))
	if ref == nil {
		t.Fatal("actor ref is nil")
	}
}

type actorFunc func(ctx actor.Context)

func (f actorFunc) Receive(ctx actor.Context) { f(ctx) }

func newMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	mustMigrateDB(db)
	return db
}
