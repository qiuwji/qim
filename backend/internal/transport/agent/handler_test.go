package agent

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

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

func setupAgentTestEnv(t *testing.T) (*gin.Engine, *actor.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := actor.NewEngine()
	hubRef := mustSpawnAgent(t, engine, "agent-hub", &testHubActor{})
	gwRef := mustSpawnAgent(t, engine, "agent-gw", NewAgentGatewayActor())

	convManager := mustSpawnAgent(t, engine, "conv-manager", &testConvActor{})
	_ = convManager

	store := &testBotStore{configs: make(map[uint64]*dal.BotConfig)}
	tools := NewToolRouter(
		service.NewConvService(engine, func(id uint64) actor.Actor { return &testConvActor{convID: id} }),
		service.NewMsgService(engine),
		service.NewFriendService(engine),
		service.NewUserService(engine, func(id uint64) actor.Actor { return &testUserActor{} }),
		store,
		hubRef,
		gwRef,
	)
	subscribe := NewSubscribeRouter(hubRef)
	dispatcher := NewAgentDispatcher(tools, subscribe, hubRef, gwRef, testToken)

	router := gin.New()
	router.POST("/agent/mcp", dispatcher.HandlePost)
	router.GET("/agent/mcp", dispatcher.HandleGet)

	return router, engine
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
}

func TestAgentDispatcher_ToolsList_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentDispatcher_SubscribeEvents_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"subscribe_events","arguments":{"bot_uid":100,"events":["message_sent"]}}}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentDispatcher_InvalidEvent_BitsUT(t *testing.T) {
	router, _ := setupAgentTestEnv(t)

	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"subscribe_events","arguments":{"bot_uid":100,"events":["invalid_event"]}}}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
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

	body := `{"jsonrpc":"2.0","id":6,"method":"unknown"}`
	req := httptest.NewRequest("POST", "/agent/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
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
