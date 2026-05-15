package http

import (
	"strconv"

	"qim/internal/domain/message"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	svc *service.MsgService
}

func NewMessageHandler(svc *service.MsgService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func handleMsgResult(c *gin.Context, r message.Result, err error) {
	if err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	if r.Err != nil {
		resp.Fail(c, 400, r.Err.Error())
		return
	}
	resp.OK(c, r.Data)
}

func (h *MessageHandler) List(c *gin.Context) {
	convID := c.GetUint64("id")
	beforeSeq := c.GetInt("seq")
	limit := 20
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		limit = l
	}
	r, err := h.svc.Ask(message.ListMessagesCmd{
		ConversationID: convID,
		BeforeSeq:      int64(beforeSeq),
		Limit:          limit,
	})
	handleMsgResult(c, r, err)
}

func (h *MessageHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	convID := c.GetUint64("conversation_id")
	limit := 20
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		limit = l
	}
	r, err := h.svc.Ask(message.SearchMessagesCmd{
		ConversationID: convID,
		Keyword:        keyword,
		Limit:          limit,
	})
	handleMsgResult(c, r, err)
}
