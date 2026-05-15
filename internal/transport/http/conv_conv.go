package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *ConversationHandler) List(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.askManager(conversation.ListUserConversationsCmd{UID: uid})
	handleResult(c, r, err)
}

func (h *ConversationHandler) CreatePrivate(c *gin.Context) {
	var req struct {
		UID uint64 `json:"uid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	uid := c.GetUint64("uid")
	r, err := h.askManager(conversation.CreatePrivateConvCmd{UID1: uid, UID2: req.UID})
	handleResult(c, r, err)
}

func (h *ConversationHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name    string   `json:"name"`
		Avatar  string   `json:"avatar"`
		Members []uint64 `json:"members"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	uid := c.GetUint64("uid")
	r, err := h.askManager(conversation.CreateGroupConvCmd{
		OwnerID: uid,
		Name:    req.Name,
		Avatar:  req.Avatar,
		Members: req.Members,
	})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Delete(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	r, err := h.askConv(convID, conversation.DeleteConvCmd{UID: uid})
	handleResult(c, r, err)
}

func (h *ConversationHandler) UpdateInfo(c *gin.Context) {
	convID := c.GetUint64("id")
	var req struct {
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.askConv(convID, conversation.UpdateConvInfoCmd{Name: req.Name, Avatar: req.Avatar})
	handleResult(c, r, err)
}
