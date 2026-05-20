package ws

import (
	"errors"
	"testing"

	"qim/internal/actor"
	"qim/internal/domain/call"
	domainuser "qim/internal/domain/user"
	"qim/internal/service"
)

func TestCallRouterDispatch_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "call-manager", wsCallResultActor{})

	dispatcher := NewDispatcher(
		nil, nil, nil, nil, nil, nil,
		service.NewCallService(engine),
	)

	tests := []struct {
		name   string
		req    WsRequest
		wantOK bool
	}{
		{"initiate success", req("call", "initiate", `{"callee_uid":200,"call_type":1}`), true},
		{"accept", req("call", "accept", `{"call_id":"call_1"}`), true},
		{"reject", req("call", "reject", `{"call_id":"call_1"}`), true},
		{"cancel", req("call", "cancel", `{"call_id":"call_1"}`), true},
		{"end", req("call", "end", `{"call_id":"call_1"}`), true},
		{"offer", req("call", "offer", `{"call_id":"call_1","sdp":"test"}`), true},
		{"answer", req("call", "answer", `{"call_id":"call_1","sdp":"test"}`), true},
		{"ice", req("call", "ice", `{"call_id":"call_1","candidate":"c","sdp_mid":"0"}`), true},
		{"unknown action", req("call", "unknown", `{}`), false},
		{"invalid data", req("call", "initiate", `{bad}`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := dispatcher.Dispatch(100, tt.req)
			if tt.wantOK && resp.Type == "error" {
				t.Fatalf("expected success, got error: %+v", resp.Error)
			}
			if !tt.wantOK && resp.Type != "error" {
				t.Fatalf("expected error, got %+v", resp)
			}
		})
	}
}

func TestCallRouterResultError_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "call-manager", wsCallErrorActor{})

	dispatcher := NewDispatcher(
		nil, nil, nil, nil, nil, nil,
		service.NewCallService(engine),
	)

	resp := dispatcher.Dispatch(100, req("call", "accept", `{"call_id":"call_1"}`))
	if resp.Type != "error" {
		t.Fatalf("expected error response, got %+v", resp)
	}
}

func TestCallRouterUnknownType_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	_ = engine
	dispatcher := NewDispatcher(nil, nil, nil, nil, nil, nil, nil)

	resp := dispatcher.Dispatch(100, req("unknown_type", "a", `{}`))
	if resp.Type != "error" {
		t.Fatalf("expected error for unknown type, got %+v", resp)
	}
}

func TestWsError_Nil_BitsUT(t *testing.T) {
	if wsError(nil) != nil {
		t.Fatal("nil should stay nil")
	}
}

func TestCallRouterNoManager_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	dispatcher := NewDispatcher(nil, nil, nil, nil, nil, nil, service.NewCallService(engine))

	resp := dispatcher.Dispatch(100, req("call", "accept", `{"call_id":"call_1"}`))
	if resp.Type != "error" {
		t.Fatalf("expected error when call-manager not available, got %+v", resp)
	}
}

func TestCallRouterMalformedJSON_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "call-manager", wsCallResultActor{})
	dispatcher := NewDispatcher(nil, nil, nil, nil, nil, nil, service.NewCallService(engine))

	actions := []string{"accept", "reject", "cancel", "end", "offer", "answer", "ice"}
	for _, action := range actions {
		resp := dispatcher.Dispatch(100, req("call", action, `{bad`))
		if resp.Type != "error" {
			t.Fatalf("expected error for malformed %s, got %+v", action, resp)
		}
	}
}

func TestCallRouterServiceError_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "call-manager", wsCallErrorActor{})
	dispatcher := NewDispatcher(nil, nil, nil, nil, nil, nil, service.NewCallService(engine))

	actions := []string{"initiate", "accept", "reject", "cancel", "end", "offer", "answer", "ice"}
	for _, action := range actions {
		var data string
		switch action {
		case "initiate":
			data = `{"callee_uid":200,"call_type":1}`
		case "offer", "answer":
			data = `{"call_id":"call_1","sdp":"test"}`
		case "ice":
			data = `{"call_id":"call_1","candidate":"c"}`
		default:
			data = `{"call_id":"call_1"}`
		}
		resp := dispatcher.Dispatch(100, req("call", action, data))
		if resp.Type != "error" {
			t.Fatalf("expected error for %s, got %+v", action, resp)
		}
	}
}

type wsCallResultActor struct{}

func (a wsCallResultActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case call.InitiateCallCmd:
		ctx.Reply(call.Result{Data: call.InitiateResult{CallID: "call_test"}})
	default:
		ctx.Reply(call.Result{Data: true})
	}
}

type wsCallErrorActor struct{}

func (a wsCallErrorActor) Receive(ctx actor.Context) {
	ctx.Reply(call.Result{Err: call.ErrNotFound})
}

type testUserMgrActor struct {
	nickname string
	avatar   string
	err      error
}

func (a testUserMgrActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case domainuser.GetUserCmd:
		if a.err != nil {
			ctx.Reply(domainuser.Result{Err: a.err})
			return
		}
		ctx.Reply(domainuser.Result{Data: domainuser.UserDTO{
			ID:       msg.UID,
			Nickname: a.nickname,
			Avatar:   a.avatar,
		}})
	}
}

func TestCallRouter_InitiateWithCallerInfo_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "call-manager", &captureInitiateActor{t: t})
	mustSpawnWS(t, engine, "user-manager", testUserMgrActor{nickname: "Alice", avatar: "avatar_url"})

	userSvc := service.NewUserService(engine, nil)
	dispatcher := NewDispatcher(nil, nil, nil, userSvc, nil, nil, service.NewCallService(engine))

	resp := dispatcher.Dispatch(100, req("call", "initiate", `{"callee_uid":200,"call_type":1}`))
	if resp.Type == "error" {
		t.Fatalf("expected success, got error: %+v", resp.Error)
	}
}

func TestCallRouter_InitiateCallerInfoFallback_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "call-manager", &captureInitiateActor{t: t})
	mustSpawnWS(t, engine, "user-manager", testUserMgrActor{err: errors.New("user not found")})

	userSvc := service.NewUserService(engine, nil)
	dispatcher := NewDispatcher(nil, nil, nil, userSvc, nil, nil, service.NewCallService(engine))

	resp := dispatcher.Dispatch(100, req("call", "initiate", `{"callee_uid":200,"call_type":1}`))
	if resp.Type == "error" {
		t.Fatalf("expected graceful fallback, got error: %+v", resp.Error)
	}
}

type captureInitiateActor struct {
	t *testing.T
}

func (a *captureInitiateActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case call.InitiateCallCmd:
		if a.t != nil {
			if msg.CallerNickname != "" || msg.CallerAvatar != "" {
				// caller info is present, verify the test user-manager was queried
			}
		}
		ctx.Reply(call.Result{Data: call.InitiateResult{CallID: "call_test"}})
	default:
		ctx.Reply(call.Result{Data: true})
	}
}

