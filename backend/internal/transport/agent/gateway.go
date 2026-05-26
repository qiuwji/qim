package agent

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"qim/internal/actor"

	"go.uber.org/zap"
)

const (
	agentHeartbeat   = 30 * time.Second
	agentDisconnectTimeout = 24 * time.Hour
)

type AgentGatewayActor struct {
	sseWriter     io.Writer
	sseMu         sync.Mutex
	flusher       http.Flusher
	sseReady      bool
	pendingEvents []PushNotificationCmd
	timeoutTimer  *actor.Timer
	hubRef        *actor.ActorRef
	lastEventID   string
	rateLimitSec  int64
	rateLimitCnt  int
	rateLimitMu   sync.Mutex
}

func NewAgentGatewayActor() *AgentGatewayActor {
	return &AgentGatewayActor{}
}

func (g *AgentGatewayActor) OnStart(ctx actor.Context) {
	ctx.Self().Tell(AgentRefResolved{GwRef: ctx.Self()})
}

func (g *AgentGatewayActor) OnStop(_ actor.Context) {
	g.sseMu.Lock()
	defer g.sseMu.Unlock()
	g.sseWriter = nil
	g.sseReady = false
}

func (g *AgentGatewayActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case PushNotificationCmd:
		g.handlePush(msg)
	case SSEConnected:
		g.handleSSEConnected(ctx)
	case SSEDisconnected:
		g.handleSSEDisconnected(ctx)
	}
}

func (g *AgentGatewayActor) handlePush(msg PushNotificationCmd) {
	g.sseMu.Lock()
	defer g.sseMu.Unlock()

	if g.sseReady && g.sseWriter != nil {
		g.writeSSE(msg)
	} else {
		g.pendingEvents = append(g.pendingEvents, msg)
	}
}

func (g *AgentGatewayActor) handleSSEConnected(_ actor.Context) {
	g.sseMu.Lock()
	g.sseReady = true
	if g.timeoutTimer != nil {
		g.timeoutTimer.Cancel()
		g.timeoutTimer = nil
	}
	pending := make([]PushNotificationCmd, len(g.pendingEvents))
	copy(pending, g.pendingEvents)
	g.pendingEvents = nil
	g.sseMu.Unlock()

	for _, evt := range pending {
		g.sseMu.Lock()
		g.writeSSE(evt)
		g.sseMu.Unlock()
	}
}

func (g *AgentGatewayActor) handleSSEDisconnected(ctx actor.Context) {
	g.sseMu.Lock()
	g.sseReady = false
	g.sseMu.Unlock()

	if ctx != nil {
		g.timeoutTimer = ctx.ScheduleAfter(agentDisconnectTimeout, func() {
			zap.L().Info("agent gateway idle timeout, stopping")
		})
	}
}

func (g *AgentGatewayActor) SetupSSE(w io.Writer, flusher http.Flusher) {
	g.sseMu.Lock()
	defer g.sseMu.Unlock()
	g.sseWriter = w
	g.flusher = flusher
}

func (g *AgentGatewayActor) writeSSE(msg PushNotificationCmd) {
	if g.sseWriter == nil {
		return
	}
	data := fmt.Sprintf(`{"jsonrpc":"2.0","method":"notifications/message","params":{"bot_uid":%d,"type":"%s"}}`,
		msg.BotUID, msg.Type)
	fmt.Fprintf(g.sseWriter, "event: message\ndata: %s\n\n", data)
	if g.flusher != nil {
		g.flusher.Flush()
	}
}

func (g *AgentGatewayActor) WriteHeartbeat() {
	g.sseMu.Lock()
	defer g.sseMu.Unlock()
	if g.sseWriter != nil {
		fmt.Fprintf(g.sseWriter, ": heartbeat\n\n")
		if g.flusher != nil {
			g.flusher.Flush()
		}
	}
}

func (g *AgentGatewayActor) CheckRate() bool {
	g.rateLimitMu.Lock()
	defer g.rateLimitMu.Unlock()
	now := time.Now().Unix()
	if now != g.rateLimitSec {
		g.rateLimitSec = now
		g.rateLimitCnt = 0
	}
	if g.rateLimitCnt >= 100 {
		return false
	}
	g.rateLimitCnt++
	return true
}
