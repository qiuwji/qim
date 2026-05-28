package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"

	"go.uber.org/zap"
)

const (
	agentHeartbeat         = 30 * time.Second
	agentDisconnectTimeout = 24 * time.Hour
)

type AgentGatewayActor struct {
	sessions map[string]*gatewaySessionState // key is sessionID, value is runtime connection state.
	hubRef   *actor.ActorRef
}

type gatewaySessionState struct {
	sseWriter     io.Writer
	flusher       http.Flusher
	sseReady      bool
	pendingEvents []PushNotificationCmd
	timeoutTimer  *actor.Timer
	lastEventID   string
	rateLimitSec  int64
	rateLimitCnt  int
	sendLimitSec  map[uint64]int64
	sendLimitCnt  map[uint64]int
}

func NewAgentGatewayActor() *AgentGatewayActor {
	return &AgentGatewayActor{
		sessions: make(map[string]*gatewaySessionState),
	}
}

func (g *AgentGatewayActor) OnStart(ctx actor.Context) {
	ctx.Self().Tell(AgentRefResolved{GwRef: ctx.Self()})
}

func (g *AgentGatewayActor) OnStop(_ actor.Context) {
	g.sessions = make(map[string]*gatewaySessionState)
}

func (g *AgentGatewayActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case PushNotificationCmd:
		g.handlePush(msg)
	case SetSessionCmd:
		g.handleSetSession(msg.SessionID)
	case SetupSSECmd:
		g.setupSSE(msg.SessionID, msg.Writer, msg.Flusher)
	case SSEConnected:
		g.handleSSEConnected(msg.SessionID)
	case SSEHeartbeat:
		g.writeHeartbeat(msg.SessionID)
	case SSEDisconnected:
		g.handleSSEDisconnected(msg.SessionID, ctx)
	case RateLimitQuery:
		_ = ctx.Reply(g.allow(msg.SessionID, msg.BotUID, msg.ToolName))
	case AgentIdleTimeout:
		g.handleSessionTimeout(msg.SessionID)
	}
}

func (g *AgentGatewayActor) handlePush(msg PushNotificationCmd) {
	state := g.ensureSession(msg.SessionID)
	if state.sseReady && state.sseWriter != nil {
		g.writeSSE(state, msg)
	} else {
		state.pendingEvents = append(state.pendingEvents, msg)
	}
}

func (g *AgentGatewayActor) handleSetSession(sessionID string) {
	g.ensureSession(sessionID)
}

func (g *AgentGatewayActor) handleSSEConnected(sessionID string) {
	state := g.ensureSession(sessionID)
	state.sseReady = true
	if state.timeoutTimer != nil {
		state.timeoutTimer.Cancel()
		state.timeoutTimer = nil
	}
	pending := make([]PushNotificationCmd, len(state.pendingEvents))
	copy(pending, state.pendingEvents)
	state.pendingEvents = nil

	for _, evt := range pending {
		g.writeSSE(state, evt)
	}
}

func (g *AgentGatewayActor) handleSSEDisconnected(sessionID string, ctx actor.Context) {
	state := g.ensureSession(sessionID)
	state.sseReady = false
	state.sseWriter = nil
	state.flusher = nil

	if ctx != nil {
		state.timeoutTimer = ctx.ScheduleAfter(agentDisconnectTimeout, AgentIdleTimeout{SessionID: sessionID})
	}
}

func (g *AgentGatewayActor) setupSSE(sessionID string, w io.Writer, flusher http.Flusher) {
	state := g.ensureSession(sessionID)
	state.sseWriter = w
	state.flusher = flusher
}

func (g *AgentGatewayActor) writeSSE(state *gatewaySessionState, msg PushNotificationCmd) {
	if state == nil || state.sseWriter == nil {
		return
	}
	data, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/message",
		"params":  notificationParams(msg),
	})
	if err != nil {
		zap.L().Error("marshal agent notification failed", zap.Error(err))
		return
	}
	fmt.Fprintf(state.sseWriter, "event: message\ndata: %s\n\n", data)
	if state.flusher != nil {
		state.flusher.Flush()
	}
}

func notificationParams(msg PushNotificationCmd) map[string]any {
	params := map[string]any{
		"bot_uid": msg.BotUID,
		"type":    msg.Type,
	}
	switch e := msg.Event.(type) {
	case conversation.MessageSentEvent:
		params["message_id"] = e.MessageID
		params["conversation_id"] = e.ConversationID
		params["conv_type"] = e.ConvType
		params["seq"] = e.Seq
		params["sender_id"] = e.SenderID
		params["member_uids"] = e.MemberUIDs
		params["msg_type"] = e.MsgType
		params["content"] = e.Content
		params["reply_to"] = e.ReplyTo
		params["client_id"] = e.ClientID
		params["created_at"] = e.CreatedAt
		params["mention_uids"] = e.MentionUIDs
		params["mention_all"] = e.MentionAll
	default:
		if b, err := json.Marshal(e); err == nil {
			var eventMap map[string]any
			if err := json.Unmarshal(b, &eventMap); err == nil {
				for k, v := range eventMap {
					params[k] = v
				}
			}
		}
	}
	return params
}

func (g *AgentGatewayActor) writeHeartbeat(sessionID string) {
	state := g.ensureSession(sessionID)
	if state.sseWriter != nil {
		fmt.Fprintf(state.sseWriter, ": heartbeat\n\n")
		if state.flusher != nil {
			state.flusher.Flush()
		}
	}
}

func (g *AgentGatewayActor) allow(sessionID string, botUID uint64, toolName string) bool {
	state := g.ensureSession(sessionID)
	now := time.Now().Unix()
	if now != state.rateLimitSec {
		state.rateLimitSec = now
		state.rateLimitCnt = 0
	}
	if state.rateLimitCnt >= 100 {
		return false
	}
	state.rateLimitCnt++
	if toolName != "send_message" || botUID == 0 {
		return true
	}
	if state.sendLimitSec[botUID] != now {
		state.sendLimitSec[botUID] = now
		state.sendLimitCnt[botUID] = 0
	}
	if state.sendLimitCnt[botUID] >= 10 {
		return false
	}
	state.sendLimitCnt[botUID]++
	return true
}

func (g *AgentGatewayActor) handleSessionTimeout(sessionID string) {
	delete(g.sessions, sessionID)
}

func (g *AgentGatewayActor) ensureSession(sessionID string) *gatewaySessionState {
	if sessionID == "" {
		sessionID = "__default__"
	}
	if g.sessions[sessionID] == nil {
		g.sessions[sessionID] = &gatewaySessionState{
			sendLimitSec: make(map[uint64]int64),
			sendLimitCnt: make(map[uint64]int),
		}
	}
	return g.sessions[sessionID]
}
