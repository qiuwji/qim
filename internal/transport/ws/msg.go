package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/domain/conversation"
	"qim/internal/domain/message"
	"qim/internal/service"
)

type msgRouter struct {
	msgSvc  *service.MsgService
	convSvc *service.ConvService
}

func (r *msgRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	if action == "send" {
		return r.dispatchSend(uid, action, data)
	}
	cmd, fn, err := r.resolveAction(uid, action, data)
	if err != nil {
		return errReply(action, err.Error())
	}
	ref, err := r.msgSvc.Ref()
	if err != nil {
		return errReply(action, err.Error())
	}
	return fn(ref, cmd, action)
}

func (r *msgRouter) dispatchSend(uid uint64, action string, data json.RawMessage) WsResponse {
	var req struct {
		ConversationID uint64 `json:"conversation_id"`
		MsgType        int8   `json:"msg_type"`
		Content        string `json:"content"`
		ReplyTo        uint64 `json:"reply_to"`
		ClientID       string `json:"client_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return errReply(action, err.Error())
	}
	ref, err := r.convSvc.ConvRef(req.ConversationID)
	if err != nil {
		return errReply(action, err.Error())
	}
	return askDispatch(ref, conversation.SendMessageCmd{
		SenderID: uid,
		MsgType:  req.MsgType,
		Content:  req.Content,
		ReplyTo:  req.ReplyTo,
		ClientID: req.ClientID,
	}, action)
}

func (r *msgRouter) resolveAction(_ uint64, action string, data json.RawMessage) (any, dispatchFn, error) {
	switch action {
	case "list":
		var req struct {
			ConversationID uint64 `json:"conversation_id"`
			BeforeSeq      int64  `json:"before_seq"`
			Limit          int    `json:"limit"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		if req.Limit == 0 {
			req.Limit = 20
		}
		return message.ListMessagesCmd{
			ConversationID: req.ConversationID,
			BeforeSeq:      req.BeforeSeq,
			Limit:          req.Limit,
		}, askDispatch, nil
	case "search":
		var req struct {
			ConversationID uint64 `json:"conversation_id"`
			Keyword        string `json:"keyword"`
			Limit          int    `json:"limit"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		if req.Limit == 0 {
			req.Limit = 20
		}
		return message.SearchMessagesCmd{
			ConversationID: req.ConversationID,
			Keyword:        req.Keyword,
			Limit:          req.Limit,
		}, askDispatch, nil
	default:
		return nil, nil, fmt.Errorf("unknown msg action: %s", action)
	}
}
