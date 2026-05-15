package main

import (
	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/presence"
	"qim/internal/domain/user"
	"qim/internal/eventbus"
	"qim/internal/service"
	httphandler "qim/internal/transport/http"
	"qim/internal/transport/ws"

	"gorm.io/gorm"
)

type stores struct {
	conv   conversation.Store
	user   dal.UserStore
	msg    dal.MsgStore
	friend dal.FriendStore
}

type svcs struct {
	conv   *service.ConvService
	user   *service.UserService
	msg    *service.MsgService
	friend *service.FriendService
}

func initEngine() *actor.Engine {
	return actor.NewEngine(
		actor.WithMetrics(actor.NewDefaultMetrics()),
	)
}

func initEventBus() eventbus.Bus {
	return eventbus.NewLocalBus(1024)
}

func initDB() *gorm.DB {
	return nil
}

func initStores(db *gorm.DB) stores {
	return stores{
		conv:   dal.NewConvStore(db),
		user:   dal.NewUserStore(db),
		msg:    dal.NewMsgStore(db),
		friend: dal.NewFriendStore(db),
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
	}
}

func initActors(engine *actor.Engine, s stores, events eventbus.Bus) {
	engine.Spawn("conv-manager", conversation.NewManagerActor(s.conv, engine, events))
	engine.Spawn("user-manager", user.NewManagerActor(s.user, engine))
	engine.Spawn("msg-store", message.NewMessageStoreActor(s.msg, engine))
	engine.Spawn("friend-manager", friend.NewManagerActor(s.friend, engine))
	engine.Spawn("presence", presence.NewPresenceActor(engine))
}

func initEventHandlers(engine *actor.Engine, events eventbus.Bus) {
	presenceRef, ok := engine.Lookup("presence")
	if !ok {
		return
	}
	events.Subscribe(conversation.EventMessageSent, ws.NewMessagePushHandler(presenceRef))
}

func initHandlers(s svcs) *httphandler.Handlers {
	return &httphandler.Handlers{
		Conv:   httphandler.NewConversationHandler(s.conv),
		User:   httphandler.NewUserHandler(s.user),
		Msg:    httphandler.NewMessageHandler(s.msg),
		Friend: httphandler.NewFriendHandler(s.friend),
	}
}

func initDispatcher(s svcs) *ws.Dispatcher {
	return ws.NewDispatcher(s.conv, s.msg, s.friend, s.user)
}
