package agent

import (
	"encoding/json"
	"errors"
	"net/http"

	"qim/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeJSONRPCResult(c *gin.Context, id any, result any) {
	c.JSON(http.StatusOK, jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeJSONRPCError(c *gin.Context, status int, id any, code int, message string) {
	c.JSON(status, jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	})
}

func writeJSONRPCAppError(c *gin.Context, id any, err error) {
	code := -32603
	msg := "internal error"
	if apperr.Is(err) {
		var appErr *apperr.Error
		if errors.As(err, &appErr) {
			code = toJSONRPCCode(string(appErr.Code))
		}
		msg = err.Error()
	}
	writeJSONRPCError(c, http.StatusOK, id, code, msg)
}

var jsonRPCErrorCodes = map[string]int{
	"agent.unauthorized":      -32001,
	"agent.permission_denied": -32002,
	"agent.rate_limited":      -32003,
	"agent.bot_not_found":     -32602,
	"agent.invalid_event":     -32602,
}

func toJSONRPCCode(code string) int {
	if c, ok := jsonRPCErrorCodes[code]; ok {
		return c
	}
	return -32603
}
