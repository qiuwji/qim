package http

import (
	"strconv"

	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func paramUint(c *gin.Context, name string) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		resp.Fail(c, badRequest(err))
		return 0, false
	}
	return value, true
}

func queryUint(c *gin.Context, name string) uint64 {
	value, _ := strconv.ParseUint(c.Query(name), 10, 64)
	return value
}

func queryInt64(c *gin.Context, name string) int64 {
	value, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return value
}
