package gateway

import (
	"github.com/gorilla/websocket"
	"qim/internal/actor"
)

type GatewayActor struct {
	conn       *websocket.Conn
	sessionRef *actor.ActorRef
	uid        uint64
	engine     *actor.Engine
}

func NewGatewayActor(conn *websocket.Conn, engine *actor.Engine) *GatewayActor {
	return &GatewayActor{conn: conn, engine: engine}
}

func (a *GatewayActor) Receive(ctx actor.Context) {}
