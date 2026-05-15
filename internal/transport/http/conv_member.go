package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *ConversationHandler) Members(c *gin.Context) {
	convID := c.GetUint64("id")
	r, err := h.svc.AskConv(convID, conversation.ListMembersQuery{})
	handleResult(c, r, err)
}

func (h *ConversationHandler) AddMember(c *gin.Context) {
	convID := c.GetUint64("id")
	var req struct {
		UID  uint64 `json:"uid"`
		Role int8   `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.AskConv(convID, conversation.AddMemberCmd{UID: req.UID, Role: conversation.MemberRole(req.Role)})
	handleResult(c, r, err)
}

func (h *ConversationHandler) RemoveMember(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	r, err := h.svc.AskConv(convID, conversation.RemoveMemberCmd{UID: uid})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Leave(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	r, err := h.svc.AskConv(convID, conversation.LeaveConvCmd{UID: uid})
	handleResult(c, r, err)
}

func (h *ConversationHandler) SetRole(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	var req struct {
		Role int8 `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.AskConv(convID, conversation.SetRoleCmd{UID: uid, Role: conversation.MemberRole(req.Role)})
	handleResult(c, r, err)
}

func (h *ConversationHandler) TransferOwner(c *gin.Context) {
	convID := c.GetUint64("id")
	var req struct {
		NewOwnerID uint64 `json:"new_owner_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.AskConv(convID, conversation.TransferOwnerCmd{NewOwnerID: req.NewOwnerID})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Dissolve(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	r, err := h.svc.AskConv(convID, conversation.DissolveConvCmd{OwnerID: uid})
	handleResult(c, r, err)
}
