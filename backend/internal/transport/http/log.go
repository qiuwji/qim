package http

import (
	"qim/internal/pkg/logx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func logHTTPError(c *gin.Context, msg string, err error) {
	if err == nil {
		return
	}
	logx.FromContext(c.Request.Context()).Warn(
		msg,
		zap.String("method", c.Request.Method),
		zap.String("path", c.FullPath()),
		zap.Error(err),
	)
}
