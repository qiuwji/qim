package ws

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/service"
)

const askTimeout = 5 * time.Second

type WsDispatcher interface {
	Dispatch(uid uint64, req WsRequest) WsResponse
}

type dispatchFn func(ref *actor.ActorRef, cmd any, action string) WsResponse

func askDispatch(ref *actor.ActorRef, cmd any, action string) WsResponse {
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return WsResponse{Type: "error", Action: action, Data: err.Error()}
	}
	return WsResponse{Type: "ack", Action: action, Data: raw}
}

func tellDispatch(ref *actor.ActorRef, cmd any, action string) WsResponse {
	ref.Tell(cmd)
	return WsResponse{Type: "ack", Action: action}
}

type Dispatcher struct {
	conv   *convRouter
	msg    *msgRouter
	friend *friendRouter
	user   *userRouter
}

func NewDispatcher(
	convSvc *service.ConvService,
	msgSvc *service.MsgService,
	friendSvc *service.FriendService,
	userSvc *service.UserService,
) *Dispatcher {
	return &Dispatcher{
		conv:   &convRouter{svc: convSvc},
		msg:    &msgRouter{svc: msgSvc},
		friend: &friendRouter{svc: friendSvc},
		user:   &userRouter{svc: userSvc},
	}
}

func (d *Dispatcher) Dispatch(uid uint64, req WsRequest) WsResponse {
	switch req.Type {
	case "conv":
		return d.conv.dispatch(uid, req.Action, req.Data)
	case "msg":
		return d.msg.dispatch(uid, req.Action, req.Data)
	case "friend":
		return d.friend.dispatch(uid, req.Action, req.Data)
	case "user":
		return d.user.dispatch(uid, req.Action, req.Data)
	default:
		return WsResponse{Type: "error", Data: fmt.Sprintf("unknown type: %s", req.Type)}
	}
}

func errReply(action string, msg string) WsResponse {
	return WsResponse{Type: "error", Action: action, Data: msg}
}
