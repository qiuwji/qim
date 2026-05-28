package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"qim/internal/dal"
	"qim/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type mcpMethodSpec struct {
	requireSession bool
	rateLimited    bool
	handle         func(sessionID string, params json.RawMessage) (any, error)
	after          func(c *gin.Context, result any) any
}

func (d *AgentDispatcher) Dispatch(method string, sessionID string, params json.RawMessage) (any, error) {
	return d.resolveMethod(method).handle(sessionID, params)
}

func (d *AgentDispatcher) resolveMethod(method string) mcpMethodSpec {
	if spec, ok := d.methodSpecs()[method]; ok {
		return spec
	}
	return mcpMethodSpec{
		requireSession: true,
		handle: func(_ string, _ json.RawMessage) (any, error) {
			return nil, unknownMethodError(method)
		},
	}
}

func (d *AgentDispatcher) methodSpecs() map[string]mcpMethodSpec {
	return map[string]mcpMethodSpec{
		"initialize": {
			handle: d.handleInitialize,
			after:  attachSessionHeader,
		},
		"notifications/initialized": {
			requireSession: true,
			handle:         d.handleInitialized,
		},
		"tools/list": {
			requireSession: true,
			handle:         d.handleToolsList,
		},
		"tools/call": {
			requireSession: true,
			rateLimited:    true,
			handle:         d.handleToolCall,
		},
	}
}

func (d *AgentDispatcher) handleInitialize(_ string, _ json.RawMessage) (any, error) {
	sessionID := newSessionID()
	if d.agentStore != nil {
		if err := d.agentStore.CreateSession(dal.NewAgentSession(sessionID, agentDisconnectTimeout)); err != nil {
			return nil, err
		}
	}
	if d.gwRef != nil {
		d.gwRef.Tell(SetSessionCmd{SessionID: sessionID})
	}
	return map[string]any{
		"session_id":      sessionID,
		"protocolVersion": "2025-03-26",
		"capabilities": map[string]any{
			"tools": map[string]bool{"listChanged": true},
		},
		"serverInfo": map[string]string{
			"name":    "QIM-Agent",
			"version": "1.0.0",
		},
	}, nil
}

func attachSessionHeader(c *gin.Context, result any) any {
	generated, ok := result.(map[string]any)
	if !ok {
		return result
	}
	if sid, _ := generated["session_id"].(string); sid != "" {
		c.Header("Mcp-Session-Id", sid)
		delete(generated, "session_id")
	}
	return generated
}

func (d *AgentDispatcher) handleInitialized(_ string, _ json.RawMessage) (any, error) {
	return nil, nil
}

func (d *AgentDispatcher) handleToolsList(_ string, _ json.RawMessage) (any, error) {
	return map[string]any{"tools": d.tools.ListTools()}, nil
}

func unknownMethodError(method string) error {
	return apperr.New("agent.unknown_method", fmt.Sprintf("unknown method: %s", method))
}

func (d *AgentDispatcher) checkRate(sessionID string, botUID uint64, toolName string) bool {
	if d.gwRef == nil {
		return true
	}
	raw, err := d.gwRef.Ask(RateLimitQuery{SessionID: sessionID, BotUID: botUID, ToolName: toolName}, time.Second)
	if err != nil {
		return false
	}
	allowed, ok := raw.(bool)
	return ok && allowed
}

func toolRateKey(params json.RawMessage) (uint64, string) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return 0, ""
	}
	var args struct {
		BotUID uint64 `json:"bot_uid"`
	}
	_ = json.Unmarshal(call.Arguments, &args)
	return args.BotUID, call.Name
}

func (d *AgentDispatcher) handleToolCall(sessionID string, params json.RawMessage) (any, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}

	switch call.Name {
	case "subscribe_events":
		return d.subscribeAndResolve(sessionID, "subscribe_events", call.Arguments)
	case "unsubscribe_events":
		return d.subscribeAndResolve(sessionID, "unsubscribe_events", call.Arguments)
	default:
		return d.tools.Resolve(sessionID, call.Name, call.Arguments)
	}
}

func (d *AgentDispatcher) subscribeAndResolve(sessionID string, action string, raw json.RawMessage) (any, error) {
	switch action {
	case "subscribe_events":
		var p subscribeParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return d.subscribe.Resolve(sessionID, action, p)
	case "unsubscribe_events":
		var p unsubscribeParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return d.subscribe.Resolve(sessionID, action, p)
	}
	return nil, fmt.Errorf("unknown subscribe action: %s", action)
}
