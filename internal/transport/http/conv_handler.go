package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	svc *service.ConvService
}

func NewConversationHandler(svc *service.ConvService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

func handleResult(c *gin.Context, r conversation.Result, err error) {
	if err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	if r.Err != nil {
		if payload, ok := conversation.ToErrorPayload(r.Err); ok {
			resp.Fail(c, 400, payload.Message)
			return
		}
		resp.Fail(c, 400, r.Err.Error())
		return
	}
	resp.OK(c, r.Data)
}
