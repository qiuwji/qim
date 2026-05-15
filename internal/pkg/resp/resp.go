package resp

import "github.com/gin-gonic/gin"

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Response{Code: 0, Data: data})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(200, Response{Code: code, Message: msg})
}
