package main

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/presence"
	"qim/internal/domain/user"
	httphandler "qim/internal/transport/http"
	wsdispatcher "qim/internal/transport/ws"

	"gorm.io/gorm"
)

type app struct {
	engine   *actor.Engine
	db       *gorm.DB
	stores   stores
	handlers handlers
}

type stores struct {
	conv   dal.ConvStore
	user   dal.UserStore
	msg    dal.MsgStore
	friend dal.FriendStore
}

type handlers struct {
	conv   *httphandler.ConversationHandler
	user   *httphandler.UserHandler
	msg    *httphandler.MessageHandler
	friend *httphandler.FriendHandler
}

func initEngine() *actor.Engine {
	return actor.NewEngine(
		actor.WithMetrics(actor.NewDefaultMetrics()),
	)
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

func initActors(engine *actor.Engine, s stores) {
	engine.Spawn("conv-manager", conversation.NewManagerActor(s.conv, engine))
	engine.Spawn("user-manager", user.NewManagerActor(s.user, engine))
	engine.Spawn("msg-store", message.NewMessageStoreActor(s.msg, engine))
	engine.Spawn("friend-manager", friend.NewManagerActor(s.friend, engine))
	engine.Spawn("presence", presence.NewPresenceActor(engine))
}

func initHandlers(engine *actor.Engine, s stores) handlers {
	convFn := func(convID uint64) actor.Actor {
		return conversation.NewConversationActor(convID, s.conv, engine)
	}
	sessionFn := func(uid uint64) actor.Actor {
		return user.NewSessionActor(uid, s.user, engine)
	}
	return handlers{
		conv: httphandler.NewConversationHandler(engine, convFn),
		user: httphandler.NewUserHandler(engine, sessionFn),
		msg:  httphandler.NewMessageHandler(engine),
		friend: httphandler.NewFriendHandler(func(cmd any) (friend.Result, error) {
			ref, ok := engine.Lookup("friend-manager")
			if !ok {
				return friend.Result{}, fmt.Errorf("friend manager unavailable")
			}
			raw, err := ref.Ask(cmd, 5*time.Second)
			if err != nil {
				return friend.Result{}, err
			}
			r, ok := raw.(friend.Result)
			if !ok {
				return friend.Result{}, fmt.Errorf("unexpected result type")
			}
			return r, nil
		}),
	}
}

func initDispatcher(engine *actor.Engine, s stores) *wsdispatcher.Dispatcher {
	convFn := func(convID uint64) actor.Actor {
		return conversation.NewConversationActor(convID, s.conv, engine)
	}
	sessionFn := func(uid uint64) actor.Actor {
		return user.NewSessionActor(uid, s.user, engine)
	}
	return wsdispatcher.NewDispatcher(engine, convFn, sessionFn)
}
