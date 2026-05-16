package http

import (
	"qim/internal/domain/conversation"
	"qim/internal/pkg/resp"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	svc     *service.ConvService
	userSvc *service.UserService
}

func NewConversationHandler(svc *service.ConvService, userSvc *service.UserService) *ConversationHandler {
	return &ConversationHandler{svc: svc, userSvc: userSvc}
}

func handleResult(c *gin.Context, r conversation.Result, err error) {
	if err != nil {
		logHTTPError(c, "conversation request failed", err)
		resp.Fail(c, internalError(err))
		return
	}
	if r.Err != nil {
		logHTTPError(c, "conversation domain error", r.Err)
		resp.Fail(c, r.Err)
		return
	}
	resp.OK(c, r.Data)
}
