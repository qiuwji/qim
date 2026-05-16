package middleware

import (
	"time"

	"qim/internal/pkg/logx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LogID() gin.HandlerFunc {
	return func(c *gin.Context) {
		logID := c.GetHeader(logx.LogIDHeader)
		if logID == "" {
			logID = logx.NewLogID()
		}

		c.Set(logx.LogIDKey, logID)
		c.Header(logx.LogIDHeader, logID)
		c.Request = c.Request.WithContext(logx.WithLogID(c.Request.Context(), logID))

		start := time.Now()
		c.Next()

		logx.FromContext(c.Request.Context()).Info(
			"http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
