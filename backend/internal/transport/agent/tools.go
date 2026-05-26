package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/user"
	"qim/internal/service"
)

const askTimeout = 5 * time.Second

type ToolRouter struct {
	convSvc   *service.ConvService
	msgSvc    *service.MsgService
	friendSvc *service.FriendService
	userSvc   *service.UserService
	botStore  dal.BotStore
	hubRef    *actor.ActorRef
	gwRef     *actor.ActorRef
}

func NewToolRouter(
	convSvc *service.ConvService,
	msgSvc *service.MsgService,
	friendSvc *service.FriendService,
	userSvc *service.UserService,
	botStore dal.BotStore,
	hubRef *actor.ActorRef,
	gwRef *actor.ActorRef,
) *ToolRouter {
	return &ToolRouter{
		convSvc:   convSvc,
		msgSvc:    msgSvc,
		friendSvc: friendSvc,
		userSvc:   userSvc,
		botStore:  botStore,
		hubRef:    hubRef,
		gwRef:     gwRef,
	}
}

func (r *ToolRouter) Resolve(toolName string, arguments json.RawMessage) (any, error) {
	switch toolName {
	case "send_message":
		return r.sendMessage(arguments)
	case "get_conversations":
		return r.getConversations(arguments)
	case "get_messages":
		return r.getMessages(arguments)
	case "search_messages":
		return r.searchMessages(arguments)
	case "search_all_messages":
		return r.searchAllMessages(arguments)
	case "get_friends":
		return r.getFriends(arguments)
	case "get_friend_conversations":
		return r.getFriendConversations(arguments)
	case "get_group_info":
		return r.getGroupInfo(arguments)
	case "get_user":
		return r.getUser(arguments)
	case "request_approval":
		return r.requestApproval(arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (r *ToolRouter) checkPermission(botUID uint64, perm string) error {
	if r.hubRef == nil {
		return nil
	}
	return nil
}

func (r *ToolRouter) sendMessage(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID         uint64 `json:"bot_uid"`
		ConversationID uint64 `json:"conversation_id"`
		MsgType        int8   `json:"msg_type"`
		Content        string `json:"content"`
		ReplyTo        uint64 `json:"reply_to"`
		ClientID       string `json:"client_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	ref, err := r.convSvc.ConvRef(p.ConversationID)
	if err != nil {
		return nil, err
	}
	cmd := conversation.SendMessageCmd{
		SenderID:    p.BotUID,
		MsgType:     p.MsgType,
		Content:     p.Content,
		ReplyTo:     p.ReplyTo,
		ClientID:    p.ClientID,
	}
	if err := ref.Tell(cmd); err != nil {
		return nil, err
	}
	return map[string]string{"status": "sent"}, nil
}

func (r *ToolRouter) getConversations(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID uint64 `json:"bot_uid"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	result, err := r.convSvc.AskManager(conversation.ListUserConversationsCmd{UID: p.BotUID})
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Data, nil
}

func (r *ToolRouter) getMessages(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID         uint64 `json:"bot_uid"`
		ConversationID uint64 `json:"conversation_id"`
		BeforeSeq      int64  `json:"before_seq"`
		Limit          int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	ref, err := r.msgSvc.ReaderRef(p.ConversationID)
	if err != nil {
		return nil, err
	}
	raw2, err := ref.Ask(message.ListMessagesCmd{
		ConversationID: p.ConversationID,
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

func (r *ToolRouter) searchMessages(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID         uint64 `json:"bot_uid"`
		ConversationID uint64 `json:"conversation_id"`
		Keyword        string `json:"keyword"`
		Limit          int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	ref, err := r.msgSvc.ReaderRef(p.ConversationID)
	if err != nil {
		return nil, err
	}
	raw2, err := ref.Ask(message.SearchMessagesCmd{
		ConversationID: p.ConversationID,
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

func (r *ToolRouter) searchAllMessages(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID  uint64 `json:"bot_uid"`
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	convs, err := r.convSvc.AskManager(conversation.ListUserConversationsCmd{UID: p.BotUID})
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

func (r *ToolRouter) getFriends(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID uint64 `json:"bot_uid"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	result, err := r.friendSvc.Ask(friend.ListFriendsCmd{UID: p.BotUID})
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Data, nil
}

func (r *ToolRouter) getFriendConversations(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID    uint64 `json:"bot_uid"`
		FriendUID uint64 `json:"friend_uid"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return r.convSvc.AskManager(conversation.ListUserConversationsCmd{UID: p.BotUID})
}

func (r *ToolRouter) getGroupInfo(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID uint64 `json:"bot_uid"`
		ConvID uint64 `json:"conversation_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return r.convSvc.AskConv(p.ConvID, conversation.ListMembersQuery{})
}

func (r *ToolRouter) getUser(raw json.RawMessage) (any, error) {
	var p struct {
		BotUID uint64 `json:"bot_uid"`
		UID    uint64 `json:"uid"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	result, err := r.userSvc.AskManager(user.GetUserCmd{UID: p.UID})
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Data, nil
}

func (r *ToolRouter) requestApproval(raw json.RawMessage) (any, error) {
	return map[string]string{"status": "pending", "message": "approval sent to user"}, nil
}
