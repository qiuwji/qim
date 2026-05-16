package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func (h *ConversationHandler) Pin(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	uid := c.GetUint64("uid")
	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskManager(conversation.PinConvCmd{UID: uid, ConversationID: convID, Pinned: req.Pinned})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Mute(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	uid := c.GetUint64("uid")
	var req struct {
		Muted bool `json:"muted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskManager(conversation.MuteConvCmd{UID: uid, ConversationID: convID, Muted: req.Muted})
	handleResult(c, r, err)
}

func (h *ConversationHandler) Read(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	uid := c.GetUint64("uid")
	var req struct {
		Seq int64 `json:"seq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	r, err := h.svc.AskManager(conversation.ReadConvCmd{UID: uid, ConversationID: convID, Seq: req.Seq})
	handleResult(c, r, err)
}

func (h *ConversationHandler) ReadAll(c *gin.Context) {
	uid := c.GetUint64("uid")
	r, err := h.svc.AskManager(conversation.ReadAllConvCmd{UID: uid})
	handleResult(c, r, err)
}
