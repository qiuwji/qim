package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *ConversationHandler) Pin(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.AskConv(convID, conversation.PinConvCmd{UID: uid, Pinned: req.Pinned})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Mute(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	var req struct {
		Muted bool `json:"muted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.AskConv(convID, conversation.MuteConvCmd{UID: uid, Muted: req.Muted})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Read(c *gin.Context) {
	convID := c.GetUint64("id")
	uid := c.GetUint64("uid")
	var req struct {
		Seq int64 `json:"seq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	r, err := h.svc.AskConv(convID, conversation.ReadConvCmd{UID: uid, Seq: req.Seq})
	handleResult(c, r, err)
}

func (h *ConversationHandler) ReadAll(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.AskManager(conversation.ReadAllConvCmd{UID: uid})
	handleResult(c, r, err)
}
