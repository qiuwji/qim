package agent

import (
	"encoding/json"
	"fmt"

	"qim/internal/domain/conversation"
	"qim/internal/domain/message"
)

func init() {
	autoRegister((*ToolRouter).messageTools)
}

func (r *ToolRouter) messageTools() []ToolSpec {
	return []ToolSpec{
		{
			Name:         "send_message",
			Desc:         "send a message as bot",
			NeedConv:     true,
			NeedApproval: true,
			Handle:       r.handleSendMessage,
		},
		{
			Name:        "get_messages",
			Desc:        "fetch conversation messages",
			Permissions: []string{permissionMessageSearch},
			NeedConv:    true,
			Handle:      r.handleGetMessages,
		},
		{
			Name:        "search_messages",
			Desc:        "search messages in a conversation",
			Permissions: []string{permissionMessageSearch},
			NeedConv:    true,
			Handle:      r.handleSearchMessages,
		},
		{
			Name:        "search_all_messages",
			Desc:        "search messages across all conversations",
			Permissions: []string{permissionMessageSearch},
			Handle:      r.handleSearchAllMessages,
		},
	}
}

func (r *ToolRouter) handleSendMessage(ctx ToolContext, raw json.RawMessage) (any, error) {
	var p struct {
		MsgType  int8   `json:"msg_type"`
		Content  string `json:"content"`
		ReplyTo  uint64 `json:"reply_to"`
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	ref, err := r.convSvc.ConvRef(ctx.ConvID)
	if err != nil {
		return nil, err
	}
	cmd := conversation.SendMessageCmd{
		SenderID: ctx.BotUID,
		MsgType:  p.MsgType,
		Content:  p.Content,
		ReplyTo:  p.ReplyTo,
		ClientID: p.ClientID,
	}
	if err := ref.Tell(cmd); err != nil {
		return nil, err
	}
	return map[string]string{"status": "sent"}, nil
}

func (r *ToolRouter) handleGetMessages(ctx ToolContext, raw json.RawMessage) (any, error) {
	var p struct {
		BeforeSeq int64 `json:"before_seq"`
		Limit     int   `json:"limit"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	ref, err := r.msgSvc.ReaderRef(ctx.ConvID)
	if err != nil {
		return nil, err
	}
	raw2, err := ref.Ask(message.ListMessagesCmd{
		ConversationID: ctx.ConvID,
		BeforeSeq:      p.BeforeSeq,
		Limit:          p.Limit,
	}, askTimeout)
	if err != nil {
		return nil, err
	}
	if result, ok := raw2.(message.Result); ok {
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Data, nil
	}
	return raw2, nil
}

func (r *ToolRouter) handleSearchMessages(ctx ToolContext, raw json.RawMessage) (any, error) {
	var p struct {
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	ref, err := r.msgSvc.ReaderRef(ctx.ConvID)
	if err != nil {
		return nil, err
	}
	raw2, err := ref.Ask(message.SearchMessagesCmd{
		ConversationID: ctx.ConvID,
		Keyword:        p.Keyword,
		Limit:          p.Limit,
	}, askTimeout)
	if err != nil {
		return nil, err
	}
	if result, ok := raw2.(message.Result); ok {
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Data, nil
	}
	return raw2, nil
}

func (r *ToolRouter) handleSearchAllMessages(ctx ToolContext, raw json.RawMessage) (any, error) {
	var p struct {
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	convs, err := r.convSvc.AskManager(conversation.ListUserConversationsCmd{UID: ctx.BotUID})
	if err != nil {
		return nil, err
	}
	if convs.Err != nil {
		return nil, convs.Err
	}
	dtos, ok := convs.Data.([]conversation.UserConvDTO)
	if !ok {
		return nil, fmt.Errorf("unexpected conversations result type")
	}
	var results []message.MessageDTO
	for _, dto := range dtos {
		ref, err := r.msgSvc.ReaderRef(dto.ConversationID)
		if err != nil {
			continue
		}
		raw2, err := ref.Ask(message.SearchMessagesCmd{
			ConversationID: dto.ConversationID,
			Keyword:        p.Keyword,
			Limit:          p.Limit,
		}, askTimeout)
		if err != nil {
			continue
		}
		if result, ok := raw2.(message.Result); ok && result.Err == nil {
			if msgs, ok := result.Data.([]message.MessageDTO); ok {
				results = append(results, msgs...)
			}
		}
	}
	return results, nil
}
