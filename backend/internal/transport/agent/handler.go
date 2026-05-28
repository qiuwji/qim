package agent

import (
	"net/http"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"

	"github.com/gin-gonic/gin"
)

var agentSSEHeartbeatInterval = 30 * time.Second

type AgentDispatcher struct {
	// tools routes MCP tool calls to QIM domain actors/stores.
	tools *ToolRouter

	// subscribe routes event subscribe/unsubscribe tools to AgentHubActor.
	subscribe *SubscribeRouter

	// gwRef points to AgentGatewayActor, which owns SSE session state and rate limits.
	gwRef *actor.ActorRef

	// hubRef points to AgentHubActor, which owns event subscriptions and permission cache.
	hubRef *actor.ActorRef

	// agentStore persists MCP session metadata and validates session lifecycle.
	agentStore dal.AgentStore

	// auth validates the platform token used by external Agent clients.
	auth PlatformAuthenticator
}

func NewAgentDispatcher(
	tools *ToolRouter,
	subscribe *SubscribeRouter,
	hubRef *actor.ActorRef,
	gwRef *actor.ActorRef,
	agentStore dal.AgentStore,
	platformToken string,
) *AgentDispatcher {
	return &AgentDispatcher{
		tools:      tools,
		subscribe:  subscribe,
		hubRef:     hubRef,
		gwRef:      gwRef,
		agentStore: agentStore,
		auth:       NewPlatformAuthenticator(platformToken),
	}
}

func (d *AgentDispatcher) HandlePost(c *gin.Context) {
	if !d.auth.ValidHeader(c.GetHeader("Authorization")) {
		writeJSONRPCError(c, http.StatusUnauthorized, nil, -32001, "unauthorized")
		return
	}

	var req jsonRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeJSONRPCError(c, http.StatusBadRequest, nil, -32700, "parse error")
		return
	}

	spec := d.resolveMethod(req.Method)
	sessionID := c.GetHeader("Mcp-Session-Id")
	if spec.requireSession && !d.validSession(sessionID) {
		writeJSONRPCError(c, http.StatusUnauthorized, req.ID, -32001, "unauthorized")
		return
	}
	if spec.rateLimited {
		botUID, toolName := toolRateKey(req.Params)
		if !d.checkRate(sessionID, botUID, toolName) {
			writeJSONRPCError(c, http.StatusOK, req.ID, -32003, ErrRateLimited.Error())
			return
		}
	}
	if spec.requireSession {
		_ = d.touchSession(sessionID)
	}

	result, err := spec.handle(sessionID, req.Params)
	if err != nil {
		writeJSONRPCAppError(c, req.ID, err)
		return
	}
	if spec.after != nil {
		result = spec.after(c, result)
	}
	writeJSONRPCResult(c, req.ID, result)
}

func (d *AgentDispatcher) HandleGet(c *gin.Context) {
	sessionID := c.GetHeader("Mcp-Session-Id")
	if !d.auth.ValidHeader(c.GetHeader("Authorization")) || !d.validSession(sessionID) {
		c.Status(http.StatusUnauthorized)
		return
	}
	_ = d.touchSession(sessionID)

	prepareSSE(c)
	d.bindSSE(sessionID, c.Writer)
	d.runSSELoop(c, sessionID)
}

func prepareSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
}

func (d *AgentDispatcher) bindSSE(sessionID string, w gin.ResponseWriter) {
	if d.gwRef != nil {
		d.gwRef.Tell(SetupSSECmd{SessionID: sessionID, Writer: w, Flusher: w})
	}
	WriteSSEConnected(w)
	if d.gwRef != nil {
		d.gwRef.Tell(SSEConnected{SessionID: sessionID})
	}
}

func (d *AgentDispatcher) runSSELoop(c *gin.Context, sessionID string) {
	ticker := time.NewTicker(agentSSEHeartbeatInterval)
	defer ticker.Stop()
	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			d.disconnectSSE(sessionID)
			return
		case <-ticker.C:
			_ = d.touchSession(sessionID)
			d.writeSSEHeartbeat(c.Writer, sessionID)
		}
	}
}

func (d *AgentDispatcher) disconnectSSE(sessionID string) {
	if d.gwRef != nil {
		d.gwRef.Tell(SSEDisconnected{SessionID: sessionID})
	}
}

func (d *AgentDispatcher) writeSSEHeartbeat(w gin.ResponseWriter, sessionID string) {
	if d.gwRef != nil {
		d.gwRef.Tell(SSEHeartbeat{SessionID: sessionID})
		return
	}
	WriteSSEHeartbeat(w)
}
