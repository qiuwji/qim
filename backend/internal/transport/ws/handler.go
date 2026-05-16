package ws

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/user"
	"qim/internal/pkg/apperr"
	"qim/internal/service"

	"go.uber.org/zap"
)

const askTimeout = 5 * time.Second

type WsDispatcher interface {
	Dispatch(uid uint64, req WsRequest) WsResponse
}

type dispatchFn func(ref *actor.ActorRef, cmd any, action string) WsResponse

func askDispatch(ref *actor.ActorRef, cmd any, action string) WsResponse {
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return errReply(action, err)
	}
	if resp, ok := resultReply(action, raw); ok {
		return resp
	}
	return WsResponse{Type: "ack", Action: action, Data: raw}
}

func tellDispatch(ref *actor.ActorRef, cmd any, action string) WsResponse {
	if err := ref.Tell(cmd); err != nil {
		zap.L().Warn("websocket tell dispatch failed", zap.String("actor", ref.Name()), zap.String("action", action), zap.Error(err))
		return errReply(action, err)
	}
	return WsResponse{Type: "ack", Action: action}
}

type Dispatcher struct {
	conv     *convRouter
	msg      *msgRouter
	friend   *friendRouter
	user     *userRouter
	presence *presenceRouter
}

func NewDispatcher(
	convSvc *service.ConvService,
	msgSvc *service.MsgService,
	friendSvc *service.FriendService,
	userSvc *service.UserService,
	presenceRef *actor.ActorRef,
	friendRef *actor.ActorRef,
) *Dispatcher {
	return &Dispatcher{
		conv:     &convRouter{svc: convSvc},
		msg:      &msgRouter{msgSvc: msgSvc, convSvc: convSvc},
		friend:   &friendRouter{svc: friendSvc},
		user:     &userRouter{svc: userSvc},
		presence: &presenceRouter{presenceRef: presenceRef, friendRef: friendRef},
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
	case "presence":
		return d.presence.dispatch(uid, req.Action, req.Data)
	default:
		return errReply("", apperr.New(apperr.CodeInvalidRequest, fmt.Sprintf("unknown type: %s", req.Type)))
	}
}

func errReply(action string, err error) WsResponse {
	payload := apperr.ToPayload(wsError(err))
	return WsResponse{Type: "error", Action: action, Error: &payload}
}

func resultReply(action string, raw any) (WsResponse, bool) {
	switch r := raw.(type) {
	case conversation.Result:
		return domainResultReply(action, r.Data, r.Err), true
	case message.Result:
		return genericResultReply(action, r.Data, r.Err), true
	case friend.Result:
		return genericResultReply(action, r.Data, r.Err), true
	case user.Result:
		return genericResultReply(action, r.Data, r.Err), true
	default:
		return WsResponse{}, false
	}
}

func domainResultReply(action string, data any, err error) WsResponse {
	if err == nil {
		return WsResponse{Type: "ack", Action: action, Data: data}
	}
	return errReply(action, err)
}

func genericResultReply(action string, data any, err error) WsResponse {
	if err != nil {
		return errReply(action, err)
	}
	return WsResponse{Type: "ack", Action: action, Data: data}
}

func wsError(err error) error {
	if err == nil || apperr.Is(err) {
		return err
	}
	return apperr.Wrap(apperr.CodeBadRequest, err.Error(), err)
}
