package ws

import (
	"encoding/json"
	"fmt"

	"qim/internal/domain/message"
	"qim/internal/service"
)

type msgRouter struct {
	svc *service.MsgService
}

func (r *msgRouter) dispatch(uid uint64, action string, data json.RawMessage) WsResponse {
	cmd, fn, err := r.resolveAction(uid, action, data)
	if err != nil {
		return errReply(action, err.Error())
	}
	ref, err := r.svc.Ref()
	if err != nil {
		return errReply(action, err.Error())
	}
	return fn(ref, cmd, action)
}

func (r *msgRouter) resolveAction(uid uint64, action string, data json.RawMessage) (any, dispatchFn, error) {
	switch action {
	case "send":
		var req struct {
			ConversationID uint64 `json:"conversation_id"`
			MsgType        int8   `json:"msg_type"`
			Content        string `json:"content"`
			ClientID       string `json:"client_id"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return message.StoreMsgCmd{
			ConversationID: req.ConversationID,
			SenderID:       uid,
			MsgType:        message.MsgType(req.MsgType),
			Content:        req.Content,
			ClientID:       req.ClientID,
		}, askDispatch, nil
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
