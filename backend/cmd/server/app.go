package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/call"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/presence"
	"qim/internal/domain/user"
	"qim/internal/eventbus"
	jwtpkg "qim/internal/pkg/jwt"
	"qim/internal/pkg/logx"
	"qim/internal/service"
	"qim/internal/transport/agent"
	httphandler "qim/internal/transport/http"
	"qim/internal/transport/ws"

	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type stores struct {
	conv   conversation.Store
	user   dal.UserStore
	msg    dal.MsgStore
	friend dal.FriendStore
	call   call.CallStore
	bot    dal.BotStore
}

func initJWT() *jwtpkg.Manager {
	secret := os.Getenv("QIM_JWT_SECRET")
	if secret == "" {
		secret = "qim-dev-secret"
	}
	return jwtpkg.NewManager(secret, 7*24*time.Hour)
}

type svcs struct {
	conv   *service.ConvService
	user   *service.UserService
	msg    *service.MsgService
	friend *service.FriendService
	call   *service.CallService
}

func initEngine() *actor.Engine {
	return actor.NewEngine(
		actor.WithMetrics(actor.NewDefaultMetrics()),
		actor.WithLogger(logx.NewActorLogger()),
	)
}

func initEventBus(engine *actor.Engine) eventbus.Bus {
	ref, err := engine.Spawn("eventbus", eventbus.NewEventBusActor())
	if err != nil {
		panic(err)
	}
	return eventbus.NewRefBus(ref)
}

func initDB() *gorm.DB {
	dbPath := os.Getenv("QIM_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "qim.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		panic(err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	mustInitSQLite(db)
	mustMigrateDB(db)
	zap.L().Info("sqlite database initialized", zap.String("path", dbPath))
	return db
}

func mustInitSQLite(db *gorm.DB) {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
	}
	for _, pragma := range pragmas {
		if err := db.Exec(pragma).Error; err != nil {
			panic(err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxOpenConns(1)
}

func mustMigrateDB(db *gorm.DB) {
	if err := db.AutoMigrate(
		&dal.Conversation{},
		&dal.Member{},
		&dal.UserConversation{},
		&dal.User{},
		&dal.Message{},
		&dal.FriendRequest{},
		&dal.FriendGroup{},
		&dal.Friend{},
		&dal.CallRecord{},
		&dal.BotConfig{},
	); err != nil {
		panic(err)
	}
	mustCreateMessageClientIndex(db)
}

func mustCreateMessageClientIndex(db *gorm.DB) {
	var indexSQL string
	if err := db.Raw(
		"SELECT sql FROM sqlite_master WHERE type = 'index' AND name = ?",
		"idx_msg_client",
	).Scan(&indexSQL).Error; err != nil {
		panic(err)
	}

	normalizedSQL := strings.ToLower(indexSQL)
	if strings.Contains(normalizedSQL, "where client_id <> ''") ||
		strings.Contains(normalizedSQL, "where client_id != ''") {
		return
	}
	if indexSQL != "" {
		if err := db.Migrator().DropIndex(&dal.Message{}, "idx_msg_client"); err != nil {
			panic(err)
		}
	}
	if err := db.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_msg_client
ON messages(conversation_id, sender_id, client_id)
WHERE client_id <> ''
`).Error; err != nil {
		panic(err)
	}
}

func initStores(db *gorm.DB) stores {
	return stores{
		conv:   dal.NewConvStore(db),
		user:   dal.NewUserStore(db),
		msg:    dal.NewMsgStore(db),
		friend: dal.NewFriendStore(db),
		call:   dal.NewCallStore(db),
		bot:    dal.NewBotStore(db),
	}
}

func initServices(engine *actor.Engine, s stores, events eventbus.Bus) svcs {
	convFn := func(convID uint64) actor.Actor {
		return conversation.NewConversationActor(convID, s.conv, engine, events)
	}
	sessionFn := func(uid uint64) actor.Actor {
		return user.NewSessionActor(uid, s.user, engine)
	}
	return svcs{
		conv:   service.NewConvService(engine, convFn),
		user:   service.NewUserService(engine, sessionFn),
		msg:    service.NewMsgService(engine),
		friend: service.NewFriendService(engine),
		call:   service.NewCallService(engine),
	}
}

func initActors(engine *actor.Engine, s stores, events eventbus.Bus) {
	mustSpawn(engine, "conv-manager", conversation.NewManagerActor(s.conv, engine, events))
	mustSpawn(engine, "user-manager", user.NewManagerActor(s.user, engine))
	for i := 0; i < service.NumMsgReaders; i++ {
		mustSpawn(engine, fmt.Sprintf("msg-reader-%d", i), message.NewMessageStoreActor(s.msg, engine))
	}
	mustSpawn(engine, "friend-manager", friend.NewManagerActor(s.friend, engine, events))
	mustSpawn(engine, "presence", presence.NewPresenceActor(engine, events))
	mustSpawn(engine, "call-manager", call.NewCallManagerActor(engine, events, s.call, s.conv))
}

func initEventHandlers(engine *actor.Engine, events eventbus.Bus) {
	if events == nil {
		panic("event bus is nil")
	}
	presenceRef, ok := engine.Lookup("presence")
	if !ok {
		panic("presence actor not found")
	}
	friendMgrRef, ok := engine.Lookup("friend-manager")
	if !ok {
		panic("friend-manager actor not found")
	}
	pushRef, err := engine.Spawn("message-push", ws.NewMessagePushActor(presenceRef, friendMgrRef))
	if err != nil {
		panic(err)
	}
	pushEvents := []string{
		conversation.EventMessageSent,
		conversation.EventMessageRevoked,
		conversation.EventConversationUpdated,
		conversation.EventMemberJoined,
		conversation.EventMemberLeft,
		conversation.EventMemberKicked,
		conversation.EventOwnerTransferred,
		conversation.EventGroupDissolved,
		friend.EventFriendRequestCreated,
		friend.EventFriendRequestHandled,
		presence.EventUserOnline,
		presence.EventUserOffline,
		call.EventCallIncoming,
		call.EventCallCalling,
		call.EventCallAccepted,
		call.EventCallRejected,
		call.EventCallCancelled,
		call.EventCallEnded,
		call.EventCallTimeout,
		call.EventCallAnsweredElsewhere,
	}
	for _, eventName := range pushEvents {
		if err := events.Subscribe(eventName, pushRef); err != nil {
			panic(err)
		}
	}
	userManagerRef, ok := engine.Lookup("user-manager")
	if !ok {
		panic("user-manager actor not found")
	}
	if err := events.Subscribe(user.EventUserOnline, userManagerRef); err != nil {
		panic(err)
	}
}

func mustSpawn(engine *actor.Engine, name string, a actor.Actor) *actor.ActorRef {
	ref, err := engine.Spawn(name, a)
	if err != nil {
		panic(err)
	}
	return ref
}

func initHandlers(s svcs, jwt *jwtpkg.Manager, stores stores) *httphandler.Handlers {
	return &httphandler.Handlers{
		Conv:   httphandler.NewConversationHandler(s.conv, s.user),
		User:   httphandler.NewUserHandler(s.user, jwt),
		Msg:    httphandler.NewMessageHandler(s.msg),
		Friend: httphandler.NewFriendHandler(s.friend, s.user),
		File:   httphandler.NewFileHandler(),
		Bot:    httphandler.NewBotHTTPHandler(s.user, s.conv, s.friend, stores.bot, stores.user),
	}
}

func initDispatcher(s svcs, engine *actor.Engine) *ws.Dispatcher {
	presenceRef, _ := engine.Lookup("presence")
	friendRef, _ := engine.Lookup("friend-manager")
	return ws.NewDispatcher(s.conv, s.msg, s.friend, s.user, presenceRef, friendRef, s.call)
}

func initAgentDispatcher(s svcs, stores stores, engine *actor.Engine, events eventbus.Bus) *agent.AgentDispatcher {
	hubRef, err := engine.Spawn("agent-hub", agent.NewAgentHubActor(events, stores.bot))
	if err != nil {
		panic(err)
	}
	gwRef, err := engine.Spawn("agent-gw", agent.NewAgentGatewayActor())
	if err != nil {
		panic(err)
	}

	hubRef.Tell(agent.AgentRefResolved{GwRef: gwRef})

	tools := agent.NewToolRouter(s.conv, s.msg, s.friend, s.user, stores.bot, hubRef, gwRef)
	subscribe := agent.NewSubscribeRouter(hubRef)

	token := os.Getenv("AGENT_PLATFORM_TOKEN")
	if token == "" {
		zap.L().Warn("AGENT_PLATFORM_TOKEN not set, agent endpoints will be unavailable")
	}

	return agent.NewAgentDispatcher(tools, subscribe, hubRef, gwRef, token)
}
