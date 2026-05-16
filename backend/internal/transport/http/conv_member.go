package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *ConversationHandler) Members(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	r, err := h.svc.AskConv(convID, conversation.ListMembersQuery{})
	handleResult(c, r, err)
}

func (h *ConversationHandler) AddMember(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	operatorID := c.GetUint64("uid")
	var req struct {
		UID      uint64 `json:"uid"`
		Username string `json:"username"`
		Role     int8   `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	targetUID := req.UID
	if req.Username != "" {
		resolvedUID, ok := resolveUsername(c, h.userSvc, req.Username)
		if !ok {
			return
		}
		targetUID = resolvedUID
	}
	r, err := h.svc.AskConv(convID, conversation.AddMemberCmd{
		OperatorID: operatorID,
		UID:        targetUID,
		Role:       conversation.MemberRole(req.Role),
	})
	handleResult(c, r, err)
}

func (h *ConversationHandler) RemoveMember(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	operatorID := c.GetUint64("uid")
	targetUID, ok := paramUint(c, "uid")
	if !ok {
		return
	}
	r, err := h.svc.AskConv(convID, conversation.RemoveMemberCmd{OperatorID: operatorID, UID: targetUID})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Leave(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	uid := c.GetUint64("uid")
	r, err := h.svc.AskConv(convID, conversation.LeaveConvCmd{UID: uid})
	handleResult(c, r, err)
}

func (h *ConversationHandler) SetRole(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	operatorID := c.GetUint64("uid")
	targetUID, ok := paramUint(c, "uid")
	if !ok {
		return
	}
	var req struct {
		Role int8 `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskConv(convID, conversation.SetRoleCmd{
		OperatorID: operatorID,
		UID:        targetUID,
		Role:       conversation.MemberRole(req.Role),
	})
	handleResult(c, r, err)
}

func (h *ConversationHandler) TransferOwner(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	operatorID := c.GetUint64("uid")
	var req struct {
		NewOwnerID uint64 `json:"new_owner_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskConv(convID, conversation.TransferOwnerCmd{OperatorID: operatorID, NewOwnerID: req.NewOwnerID})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Dissolve(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	uid := c.GetUint64("uid")
	r, err := h.svc.AskConv(convID, conversation.DissolveConvCmd{OperatorID: uid})
	handleResult(c, r, err)
}
