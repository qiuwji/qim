package resp

import (
	"qim/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    apperr.Code `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    any         `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Response{Code: apperr.CodeOK, Data: data})
}

func Fail(c *gin.Context, err error) {
	payload := apperr.ToPayload(err)
	c.JSON(200, Response{Code: payload.Code, Message: payload.Message})
}
