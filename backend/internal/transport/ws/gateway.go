package ws

import (
	"time"

	"qim/internal/actor"
	"qim/internal/domain/presence"
	userdomain "qim/internal/domain/user"
	"qim/internal/eventbus"
	"qim/internal/pkg/logx"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const idleTimeout = 6 * time.Hour

type GatewayActor struct {
	conn       *websocket.Conn
	sessionRef *actor.ActorRef
	uid        uint64
	engine     *actor.Engine
	events     eventbus.Bus
	dispatcher WsDispatcher
	presence   *actor.ActorRef
	idleTimer  *actor.Timer
	connLogID  string
}

func NewGatewayActor(uid uint64, conn *websocket.Conn, engine *actor.Engine, events eventbus.Bus, dispatcher WsDispatcher, connLogID string) *GatewayActor {
	return &GatewayActor{uid: uid, conn: conn, engine: engine, events: events, dispatcher: dispatcher, connLogID: connLogID}
}

func (a *GatewayActor) OnStart(ctx actor.Context) {
	self := ctx.Self()
	a.registerPresence(self)
	a.publishUserOnline()
	go func() {
		for {
			var req WsRequest
			if err := a.conn.ReadJSON(&req); err != nil {
				a.logger().Info("websocket disconnected", zap.Uint64("uid", a.uid), zap.Error(err))
				self.Tell(WSDisconnected{})
				return
			}
			req.LogID = logx.NewLogID()
			self.Tell(req)
		}
	}()
	a.resetIdleTimer(ctx)
}

func (a *GatewayActor) publishUserOnline() {
	if a.events == nil || a.uid == 0 {
		return
	}
	if err := a.events.Publish(userdomain.UserOnlineEvent{UID: a.uid, At: time.Now().Unix()}); err != nil {
		a.logger().Warn("publish user online event failed", zap.Uint64("uid", a.uid), zap.Error(err))
	}
}

func (a *GatewayActor) OnStop(ctx actor.Context) {
	if a.idleTimer != nil {
		a.idleTimer.Cancel()
	}
	a.unregisterPresence(ctx.Self())
	a.conn.Close()
}

func (a *GatewayActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	// 连接断开，关闭 Actor
	case WSDisconnected:
		ctx.Self().Tell(actor.PoisonPill{})
		return
	// 空闲超时，关闭连接
	case IdleTimeout:
		ctx.Self().Tell(actor.PoisonPill{})
		return
	// 客户端发来的 WS 消息
	case WsRequest:
		logger := a.messageLogger(msg.LogID)
		logger.Info("websocket message received", zap.Uint64("uid", a.uid), zap.String("type", msg.Type), zap.String("action", msg.Action))
		resp := a.dispatcher.Dispatch(a.uid, msg)
		resp.LogID = msg.LogID
		if err := a.conn.WriteJSON(resp); err != nil {
			logger.Warn("write websocket response failed", zap.Uint64("uid", a.uid), zap.String("type", msg.Type), zap.String("action", msg.Action), zap.Error(err))
		}
	// 服务端推送消息给客户端
	case PushCmd:
		logID := logx.NewLogID()
		logger := a.messageLogger(logID)
		if err := a.conn.WriteJSON(WsResponse{Type: msg.Type, Action: msg.Action, Data: msg.Data, LogID: logID}); err != nil {
			logger.Warn("write websocket push failed", zap.Uint64("uid", a.uid), zap.String("type", msg.Type), zap.Error(err))
		}
	}
	a.resetIdleTimer(ctx)
}

func (a *GatewayActor) resetIdleTimer(ctx actor.Context) {
	if a.idleTimer != nil {
		a.idleTimer.Cancel()
	}
	a.idleTimer = ctx.ScheduleAfter(idleTimeout, IdleTimeout{})
}

func (a *GatewayActor) registerPresence(self *actor.ActorRef) {
	ref, ok := a.engine.Lookup("presence")
	if !ok {
		a.logger().Error("presence actor not found", zap.Uint64("uid", a.uid))
		return
	}
	a.presence = ref
	if err := ref.Tell(presence.UserConnected{UID: a.uid, Gateway: self}); err != nil {
		a.logger().Warn("register presence failed", zap.Uint64("uid", a.uid), zap.Error(err))
	}
}

func (a *GatewayActor) unregisterPresence(self *actor.ActorRef) {
	if a.presence == nil {
		return
	}
	if err := a.presence.Tell(presence.UserDisconnected{UID: a.uid, Gateway: self}); err != nil {
		a.logger().Warn("unregister presence failed", zap.Uint64("uid", a.uid), zap.Error(err))
	}
}

func (a *GatewayActor) logger() *zap.Logger {
	return logx.WithLogIDField(a.connLogID).With(zap.String("conn_log_id", a.connLogID))
}

func (a *GatewayActor) messageLogger(logID string) *zap.Logger {
	return logx.WithLogIDField(logID).With(zap.String("conn_log_id", a.connLogID))
}
