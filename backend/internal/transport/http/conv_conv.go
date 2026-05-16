package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *ConversationHandler) List(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.AskManager(conversation.ListUserConversationsCmd{UID: uid})
	handleResult(c, r, err)
}

func (h *ConversationHandler) CreatePrivate(c *gin.Context) {
	var req struct {
		UID      uint64 `json:"uid"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	uid := c.GetUint64("uid")
	targetUID := req.UID
	if req.Username != "" {
		resolvedUID, ok := resolveUsername(c, h.userSvc, req.Username)
		if !ok {
			return
		}
		targetUID = resolvedUID
	}
	r, err := h.svc.AskManager(conversation.CreatePrivateConvCmd{UID1: uid, UID2: targetUID})
	handleResult(c, r, err)
}

func (h *ConversationHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name      string   `json:"name"`
		Avatar    string   `json:"avatar"`
		Members   []uint64 `json:"members"`
		Usernames []string `json:"usernames"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	uid := c.GetUint64("uid")
	members := req.Members
	for _, username := range req.Usernames {
		resolvedUID, ok := resolveUsername(c, h.userSvc, username)
		if !ok {
			return
		}
		members = append(members, resolvedUID)
	}
	r, err := h.svc.AskManager(conversation.CreateGroupConvCmd{
		OwnerID: uid,
		Name:    req.Name,
		Avatar:  req.Avatar,
		Members: members,
	})
	handleResult(c, r, err)
}

func (h *ConversationHandler) UpdateInfo(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Avatar      *string `json:"avatar"`
		MemberLimit *int    `json:"member_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	uid := c.GetUint64("uid")
	r, err := h.svc.AskConv(convID, conversation.UpdateConvInfoCmd{
		OperatorID:  uid,
		Name:        req.Name,
		Avatar:      req.Avatar,
		MemberLimit: req.MemberLimit,
	})
	handleResult(c, r, err)
}
