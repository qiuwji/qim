package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"qim/internal/actor"
	"qim/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type AgentDispatcher struct {
	tools         *ToolRouter
	subscribe     *SubscribeRouter
	gwRef         *actor.ActorRef
	hubRef        *actor.ActorRef
	platformToken string
}

func NewAgentDispatcher(
	tools *ToolRouter,
	subscribe *SubscribeRouter,
	hubRef *actor.ActorRef,
	gwRef *actor.ActorRef,
	platformToken string,
) *AgentDispatcher {
	return &AgentDispatcher{
		tools:         tools,
		subscribe:     subscribe,
		hubRef:        hubRef,
		gwRef:         gwRef,
		platformToken: platformToken,
	}
}

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

func (d *AgentDispatcher) HandlePost(c *gin.Context) {
	token := extractBearer(c.GetHeader("Authorization"))
	if token != d.platformToken {
		c.JSON(http.StatusUnauthorized, jsonRPCResponse{
			JSONRPC: "2.0",
			Error:   &jsonRPCError{Code: -32001, Message: "unauthorized"},
		})
		return
	}

	var req jsonRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, jsonRPCResponse{
			JSONRPC: "2.0",
			Error:   &jsonRPCError{Code: -32700, Message: "parse error"},
		})
		return
	}

	result, err := d.Dispatch(req.Method, req.Params)
	if err != nil {
		code := -32603
		msg := "internal error"
		if apperr.Is(err) {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				code = toJSONRPCCode(string(appErr.Code))
			}
			msg = err.Error()
		}
		c.JSON(http.StatusOK, jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: code, Message: msg},
		})
		return
	}

	c.JSON(http.StatusOK, jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func (d *AgentDispatcher) HandleGet(c *gin.Context) {
	token := extractBearer(c.GetHeader("Authorization"))
	if token != d.platformToken {
		c.Status(http.StatusUnauthorized)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	w := c.Writer
	WriteSSEConnected(w)

	if d.gwRef != nil {
		d.gwRef.Tell(SSEConnected{})
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			if d.gwRef != nil {
				d.gwRef.Tell(SSEDisconnected{})
			}
			return
		case <-ticker.C:
			WriteSSEHeartbeat(w)
		}
	}
}

func (d *AgentDispatcher) Dispatch(method string, params json.RawMessage) (any, error) {
	switch {
	case method == "initialize":
		return map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities": map[string]any{
				"tools": map[string]bool{"listChanged": true},
			},
			"serverInfo": map[string]string{
				"name":    "QIM-Agent",
				"version": "1.0.0",
			},
		}, nil

	case method == "notifications/initialized":
		return nil, nil

	case method == "tools/list":
		return d.listTools(), nil

	case method == "tools/call":
		return d.handleToolCall(params)

	default:
		return nil, apperr.New("agent.unknown_method", fmt.Sprintf("unknown method: %s", method))
	}
}

func (d *AgentDispatcher) handleToolCall(params json.RawMessage) (any, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}

	switch call.Name {
	case "subscribe_events":
		return d.subscribeAndResolve("subscribe_events", call.Arguments)
	case "unsubscribe_events":
		return d.subscribeAndResolve("unsubscribe_events", call.Arguments)
	default:
		return d.tools.Resolve(call.Name, call.Arguments)
	}
}

func (d *AgentDispatcher) subscribeAndResolve(action string, raw json.RawMessage) (any, error) {
	switch action {
	case "subscribe_events":
		var p subscribeParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return d.subscribe.Resolve(action, p)
	case "unsubscribe_events":
		var p unsubscribeParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return d.subscribe.Resolve(action, p)
	}
	return nil, fmt.Errorf("unknown subscribe action: %s", action)
}

func (d *AgentDispatcher) listTools() any {
	return map[string]any{
		"tools": []map[string]any{
			toolDef("send_message", "send a message as bot"),
			toolDef("get_conversations", "list bot conversations"),
			toolDef("get_messages", "fetch conversation messages"),
			toolDef("search_messages", "search messages in a conversation"),
			toolDef("search_all_messages", "search messages across all conversations"),
			toolDef("get_friends", "get friend list"),
			toolDef("get_friend_conversations", "get friend conversation info"),
			toolDef("get_group_info", "get group chat info"),
			toolDef("get_user", "get user info"),
			toolDef("subscribe_events", "subscribe to events"),
			toolDef("unsubscribe_events", "unsubscribe from events"),
			toolDef("request_approval", "request user approval for an action"),
		},
	}
}

func toolDef(name, desc string) map[string]any {
	return map[string]any{
		"name":        name,
		"description": desc,
	}
}

func extractBearer(auth string) string {
	if !strings.HasPrefix(auth, "Bearer ") {
		return auth
	}
	return auth[7:]
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
