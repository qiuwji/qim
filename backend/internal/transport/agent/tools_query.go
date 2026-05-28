package agent

import (
	"encoding/json"

	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/user"
)

func init() {
	autoRegister((*ToolRouter).queryTools)
}

func (r *ToolRouter) queryTools() []ToolSpec {
	return []ToolSpec{
		{
			Name:        "get_conversations",
			Desc:        "list bot conversations",
			Permissions: []string{permissionConversationRead},
			Handle:      r.handleGetConversations,
		},
		{
			Name:        "get_friend_conversations",
			Desc:        "get friend conversation info",
			Permissions: []string{permissionFriendRead, permissionConversationRead},
			Handle:      r.handleGetFriendConversations,
		},
		{
			Name:        "get_group_info",
			Desc:        "get group chat info",
			Permissions: []string{permissionGroupRead},
			NeedConv:    true,
			Handle:      r.handleGetGroupInfo,
		},
		{
			Name:        "get_friends",
			Desc:        "get friend list",
			Permissions: []string{permissionFriendRead},
			Handle:      r.handleGetFriends,
		},
		{
			Name:        "get_user",
			Desc:        "get user info",
			Permissions: []string{permissionFriendRead},
			Handle:      r.handleGetUser,
		},
	}
}

func (r *ToolRouter) handleGetConversations(ctx ToolContext, _ json.RawMessage) (any, error) {
	result, err := r.convSvc.AskManager(conversation.ListUserConversationsCmd{UID: ctx.BotUID})
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Data, nil
}

func (r *ToolRouter) handleGetFriendConversations(ctx ToolContext, _ json.RawMessage) (any, error) {
	result, err := r.convSvc.AskManager(conversation.ListUserConversationsCmd{UID: ctx.BotUID})
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Data, nil
}

func (r *ToolRouter) handleGetGroupInfo(ctx ToolContext, _ json.RawMessage) (any, error) {
	return r.convSvc.AskConv(ctx.ConvID, conversation.ListMembersQuery{})
}

func (r *ToolRouter) handleGetFriends(ctx ToolContext, _ json.RawMessage) (any, error) {
	ownerUID, err := r.botOwnerUID(ctx.BotUID)
	if err != nil {
		return nil, err
	}
	result, err := r.friendSvc.Ask(friend.ListFriendsCmd{UID: ownerUID})
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Data, nil
}

func (r *ToolRouter) handleGetUser(ctx ToolContext, raw json.RawMessage) (any, error) {
	var p struct {
		UID uint64 `json:"uid"`
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
