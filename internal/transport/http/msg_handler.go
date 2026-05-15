package http

import (
	"fmt"
	"strconv"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/message"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

const msgAskTimeout = 5 * time.Second

type MessageHandler struct {
	engine *actor.Engine
}

func NewMessageHandler(engine *actor.Engine) *MessageHandler {
	return &MessageHandler{engine: engine}
}

func (h *MessageHandler) storeRef() (*actor.ActorRef, bool) {
	return h.engine.Lookup("msg-store")
}

func (h *MessageHandler) askStore(cmd any) (message.Result, error) {
	ref, ok := h.storeRef()
	if !ok {
		return message.Result{}, fmt.Errorf("message store unavailable")
	}
	raw, err := ref.Ask(cmd, msgAskTimeout)
	if err != nil {
		return message.Result{}, err
	}
	r, ok := raw.(message.Result)
	if !ok {
		return message.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (h *MessageHandler) List(c *gin.Context) {
	convID := c.GetUint64("id")
	beforeSeq := c.GetInt("seq")
	limit := 20
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		limit = l
	}

	r, err := h.askStore(message.ListMessagesCmd{
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

	r, err := h.askStore(message.SearchMessagesCmd{
		ConversationID: convID,
		Keyword:        keyword,
		Limit:          limit,
	})
	handleMsgResult(c, r, err)
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
