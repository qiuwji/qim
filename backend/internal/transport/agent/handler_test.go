package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/domain/user"
	"qim/internal/service"

	"github.com/gin-gonic/gin"
)

const testToken = "test-agent-token"

type testConvActor struct {
	convID uint64
}

func (a *testConvActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case conversation.SendMessageCmd:
		ctx.Reply(conversation.Result{Data: conversation.MessageDTO{ID: 1, ConversationID: a.convID}})
	case conversation.ListUserConversationsCmd:
		ctx.Reply(conversation.Result{Data: []conversation.UserConvDTO{}})
	case conversation.CreatePrivateConvCmd:
		ctx.Reply(conversation.Result{Data: conversation.ConversationDTO{ID: 99}})
	default:
		ctx.Reply(conversation.Result{Data: true})
	}
}

func mustSpawnAgent(t *testing.T, engine *actor.Engine, name string, a actor.Actor) *actor.ActorRef {
	ref, err := engine.Spawn(name, a)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func setupAgentTestEnvWithStore(t *testing.T) (*gin.Engine, *actor.Engine, *testAgentStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := actor.NewEngine()
	hubRef := mustSpawnAgent(t, engine, "agent-hub", &testHubActor{})
	gwRef := mustSpawnAgent(t, engine, "agent-gw", NewAgentGatewayActor())

	convManager := mustSpawnAgent(t, engine, "conv-manager", &testConvActor{})
	_ = convManager

	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	agentStore := &testAgentStore{sessions: make(map[string]*dal.AgentSession), approvals: make(map[string]*dal.AgentApproval)}
	tools := NewToolRouter(
		service.NewConvService(engine, func(id uint64) actor.Actor { return &testConvActor{convID: id} }),
		service.NewMsgService(engine),
		service.NewFriendService(engine),
		service.NewUserService(engine, func(id uint64) actor.Actor { return &testUserActor{} }),
		store,
		nil,
		hubRef,
		gwRef,
		nil,
		NewApprovalManager(agentStore),
	)
	subscribe := NewSubscribeRouter(hubRef)
	dispatcher := NewAgentDispatcher(tools, subscribe, hubRef, gwRef, agentStore, testToken)

	router := gin.New()
	router.POST("/agent/mcp", dispatcher.HandlePost)
	router.GET("/agent/mcp", dispatcher.HandleGet)

	return router, engine, agentStore
}

func setupAgentTestEnv(t *testing.T) (*gin.Engine, *actor.Engine) {
	router, engine, _ := setupAgentTestEnvWithStore(t)
	return router, engine
}

func initializeAgentSession(t *testing.T, router *gin.Engine) string {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("initialize expected 200, got %d: %s", w.Code, w.Body.String())
	}
	sessionID := w.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("expected Mcp-Session-Id")
	}
	return sessionID
}

func TestAgentDispatcher_Initialize_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in initialize response")
	}
	if w.Header().Get("Mcp-Session-Id") == "" {
		t.Fatal("expected Mcp-Session-Id header")
	}
}

func TestAgentDispatcher_ToolsList_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)
	sessionID := initializeAgentSession(t, router)

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Mcp-Session-Id", sessionID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentDispatcher_SubscribeEvents_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)
	sessionID := initializeAgentSession(t, router)

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"subscribe_events","arguments":{"bot_uid":100,"events":["message_sent"]}}}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Mcp-Session-Id", sessionID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentDispatcher_InvalidEvent_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)
	sessionID := initializeAgentSession(t, router)

	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"subscribe_events","arguments":{"bot_uid":100,"events":["invalid_event"]}}}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Mcp-Session-Id", sessionID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for invalid event")
	}
}

func TestAgentDispatcher_Unauthorized_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":5,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAgentDispatcher_UnknownMethod_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)
	sessionID := initializeAgentSession(t, router)

	body := `{"jsonrpc":"2.0","id":6,"method":"unknown"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Mcp-Session-Id", sessionID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestAgentDispatcher_MissingSession_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":8,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAgentDispatcher_MultiSession_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)
	session1 := initializeAgentSession(t, router)
	session2 := initializeAgentSession(t, router)

	for _, sessionID := range []string{session1, session2} {
		body := `{"jsonrpc":"2.0","id":9,"method":"tools/list"}`
		req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+testToken)
		req.Header.Set("Mcp-Session-Id", sessionID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("session %s expected 200, got %d", sessionID, w.Code)
		}
	}
}

func TestAgentDispatcher_SSEHeartbeatTouchesSession_BitsUT(t *testing.T) {
	router, _, agentStore := setupAgentTestEnvWithStore(t)
	sessionID := initializeAgentSession(t, router)

	oldInterval := agentSSEHeartbeatInterval
	agentSSEHeartbeatInterval = 10 * time.Millisecond
	defer func() { agentSSEHeartbeatInterval = oldInterval }()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/agent/mcp", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Mcp-Session-Id", sessionID)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(35 * time.Millisecond)
	cancel()
	<-done

	if agentStore.touchCount[sessionID] < 2 {
		t.Fatalf("expected heartbeat to touch session multiple times, got %d", agentStore.touchCount[sessionID])
	}
}

func TestAgentDispatcher_NoAuthHeader_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

type testHubActor struct{}

func (a *testHubActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case SubscribeCmd, UnsubscribeCmd:
	}
}

type testUserActor struct{}

func (a *testUserActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case user.GetUserCmd:
		ctx.Reply(user.Result{Data: user.UserDTO{ID: msg.UID, Nickname: "Test"}})
	default:
		ctx.Reply(user.Result{Data: true})
	}
}
