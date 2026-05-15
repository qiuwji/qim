package ws

import (
	"time"

	"qim/internal/actor"

	"github.com/gorilla/websocket"
)

const idleTimeout = 6 * time.Hour

type GatewayActor struct {
	conn       *websocket.Conn
	sessionRef *actor.ActorRef
	uid        uint64
	engine     *actor.Engine
	dispatcher WsDispatcher
	idleTimer  *actor.Timer
}

func NewGatewayActor(uid uint64, conn *websocket.Conn, engine *actor.Engine, dispatcher WsDispatcher) *GatewayActor {
	return &GatewayActor{uid: uid, conn: conn, engine: engine, dispatcher: dispatcher}
}

func (a *GatewayActor) OnStart(ctx actor.Context) {
	self := ctx.Self()
	go func() {
		for {
			var req WsRequest
			if err := a.conn.ReadJSON(&req); err != nil {
				self.Tell(WSDisconnected{})
				return
			}
			self.Tell(req)
		}
	}()
	a.resetIdleTimer(ctx)
}

func (a *GatewayActor) OnStop(ctx actor.Context) {
	if a.idleTimer != nil {
		a.idleTimer.Cancel()
	}
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
		resp := a.dispatcher.Dispatch(a.uid, msg)
		a.conn.WriteJSON(resp)
	// 服务端推送消息给客户端
	case PushCmd:
		a.conn.WriteJSON(WsResponse{Type: msg.Type, Data: msg.Data})
	}
	a.resetIdleTimer(ctx)
}

func (a *GatewayActor) resetIdleTimer(ctx actor.Context) {
	if a.idleTimer != nil {
		a.idleTimer.Cancel()
	}
	a.idleTimer = ctx.ScheduleAfter(idleTimeout, IdleTimeout{})
}
