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
		logHTTPError(c, "message request failed", err)
		resp.Fail(c, internalError(err))
		return
	}
	if r.Err != nil {
		logHTTPError(c, "message domain error", r.Err)
		resp.Fail(c, r.Err)
		return
	}
	resp.OK(c, r.Data)
}

func (h *MessageHandler) List(c *gin.Context) {
	convID, ok := paramUint(c, "id")
	if !ok {
		return
	}
	beforeSeq := queryInt64(c, "before_seq")
	if beforeSeq == 0 {
		beforeSeq = queryInt64(c, "seq")
	}
	limit := 20
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		limit = l
	}
	r, err := h.svc.Ask(message.ListMessagesCmd{
		ConversationID: convID,
		BeforeSeq:      beforeSeq,
		Limit:          limit,
	})
	handleMsgResult(c, r, err)
}

func (h *MessageHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	convID := queryUint(c, "conversation_id")
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
