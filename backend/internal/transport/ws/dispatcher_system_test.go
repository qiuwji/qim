package ws

import (
	"encoding/json"
	"testing"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/user"
	"qim/internal/service"
)

func TestDispatcherSystemFlow_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "conv-manager", wsConvResultActor{data: "conv"})
	mustSpawnWS(t, engine, "msg-store", wsMsgResultActor{data: "msg"})
	mustSpawnWS(t, engine, "friend-manager", wsFriendResultActor{data: "friend"})
	mustSpawnWS(t, engine, "user-manager", wsUserResultActor{data: user.UserDTO{ID: 1, Username: "alice"}})

	dispatcher := NewDispatcher(
		service.NewConvService(engine, func(convID uint64) actor.Actor { return wsConvResultActor{data: convID} }),
		service.NewMsgService(engine),
		service.NewFriendService(engine),
		service.NewUserService(engine, func(uid uint64) actor.Actor { return wsUserResultActor{data: uid} }),
		nil, nil, nil,
	)

	requests := []WsRequest{
		req("conv", "list", `{}`),
		req("conv", "create_private", `{"uid":2}`),
		req("conv", "create_group", `{"name":"g","members":[2]}`),
		req("conv", "read_all", `{}`),
		req("conv", "update_info", `{"conv_id":1,"name":"g2"}`),
		req("conv", "pin", `{"conv_id":1,"pinned":true}`),
		req("conv", "mute", `{"conv_id":1,"muted":true}`),
		req("conv", "read", `{"conv_id":1,"seq":10}`),
		req("conv", "members", `{"conv_id":1}`),
		req("conv", "add_member", `{"conv_id":1,"uid":3,"role":0}`),
		req("conv", "remove_member", `{"conv_id":1,"uid":3}`),
		req("conv", "leave", `{"conv_id":1}`),
		req("conv", "set_role", `{"conv_id":1,"uid":2,"role":1}`),
		req("conv", "transfer_owner", `{"conv_id":1,"new_owner_id":2}`),
		req("conv", "dissolve", `{"conv_id":1}`),
		req("msg", "send", `{"conversation_id":1,"msg_type":1,"content":"hi","client_id":"c1"}`),
		req("msg", "revoke", `{"conversation_id":1,"message_id":1}`),
		req("msg", "typing", `{"conversation_id":1}`),
		req("msg", "list", `{"conversation_id":1,"before_seq":10,"limit":20}`),
		req("msg", "search", `{"conversation_id":1,"keyword":"hi","limit":20}`),
		req("friend", "send_request", `{"to_uid":2,"message":"hi"}`),
		req("friend", "list_incoming", `{}`),
		req("friend", "list_outgoing", `{}`),
		req("friend", "handle_request", `{"req_id":1,"accept":true}`),
		req("friend", "delete", `{"friend_uid":2}`),
		req("friend", "list", `{}`),
		req("friend", "update_remark", `{"friend_uid":2,"remark":"bob"}`),
		req("friend", "move_group", `{"friend_uid":2,"group_id":1}`),
		req("friend", "list_groups", `{}`),
		req("friend", "create_group", `{"name":"work"}`),
		req("friend", "rename_group", `{"group_id":1,"name":"friends"}`),
		req("friend", "delete_group", `{"group_id":1}`),
		req("friend", "sort_groups", `{"groups":[{"group_id":1,"sort_order":2}]}`),
		req("user", "register", `{"username":"alice","password":"secret","nickname":"Alice"}`),
		req("user", "login", `{"username":"alice","password":"secret"}`),
		req("user", "search", `{"keyword":"ali"}`),
		req("user", "get", `{"uid":1}`),
		req("user", "profile", `{}`),
		req("user", "update_profile", `{"nickname":"A"}`),
		req("user", "change_password", `{"old_password":"a","new_password":"b"}`),
	}

	for _, request := range requests {
		resp := dispatcher.Dispatch(1, request)
		if resp.Type != "ack" {
			t.Fatalf("%s/%s resp = %+v", request.Type, request.Action, resp)
		}
	}

	if resp := dispatcher.Dispatch(1, req("unknown", "x", `{}`)); resp.Type != "error" {
		t.Fatalf("unknown type resp = %+v", resp)
	}
	if resp := dispatcher.Dispatch(1, req("conv", "bad", `{}`)); resp.Type != "error" {
		t.Fatalf("unknown action resp = %+v", resp)
	}
}

func TestDispatchSendWithMention_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawnWS(t, engine, "conv-manager-mention", wsConvResultActor{data: "conv"})

	dispatcher := NewDispatcher(
		service.NewConvService(engine, func(convID uint64) actor.Actor { return wsConvResultActor{data: convID} }),
		service.NewMsgService(engine),
		service.NewFriendService(engine),
		service.NewUserService(engine, func(uid uint64) actor.Actor { return wsUserResultActor{data: uid} }),
		nil, nil, nil,
	)

	t.Run("带mention_uids和mention_all", func(t *testing.T) {
		resp := dispatcher.Dispatch(1, req("msg", "send", `{"conversation_id":1,"msg_type":1,"content":"@all","mention_uids":[2,3],"mention_all":true}`))
		if resp.Type != "ack" {
			t.Fatalf("mention send resp = %+v", resp)
		}
	})

	t.Run("无mention字段", func(t *testing.T) {
		resp := dispatcher.Dispatch(1, req("msg", "send", `{"conversation_id":1,"msg_type":1,"content":"hello"}`))
		if resp.Type != "ack" {
			t.Fatalf("no-mention send resp = %+v", resp)
		}
	})

	t.Run("无效JSON", func(t *testing.T) {
		resp := dispatcher.Dispatch(1, req("msg", "send", `{invalid}`))
		if resp.Type != "error" {
			t.Fatalf("invalid json resp = %+v", resp)
		}
	})
}

func TestReplyHelpers_BitsUT(t *testing.T) {
	if resp, ok := resultReply("a", conversation.Result{Err: conversation.ErrNotMember}); !ok || resp.Type != "error" {
		t.Fatalf("conversation error resp=%+v ok=%v", resp, ok)
	}
	if resp, ok := resultReply("a", message.Result{Err: conversation.ErrNotMember}); !ok || resp.Type != "error" {
		t.Fatalf("message error resp=%+v ok=%v", resp, ok)
	}
	if _, ok := resultReply("a", "raw"); ok {
		t.Fatalf("plain raw should not match domain result")
	}
	if wsError(nil) != nil {
		t.Fatalf("nil ws error should stay nil")
	}
}

func req(msgType, action, raw string) WsRequest {
	return WsRequest{Type: msgType, Action: action, Data: json.RawMessage(raw)}
}

type wsConvResultActor struct{ data any }

func (a wsConvResultActor) Receive(ctx actor.Context) { ctx.Reply(conversation.Result{Data: a.data}) }

type wsMsgResultActor struct{ data any }

func (a wsMsgResultActor) Receive(ctx actor.Context) { ctx.Reply(message.Result{Data: a.data}) }

type wsFriendResultActor struct{ data any }

func (a wsFriendResultActor) Receive(ctx actor.Context) { ctx.Reply(friend.Result{Data: a.data}) }

type wsUserResultActor struct{ data any }

func (a wsUserResultActor) Receive(ctx actor.Context) { ctx.Reply(user.Result{Data: a.data}) }

func mustSpawnWS(t *testing.T, engine *actor.Engine, name string, a actor.Actor) {
	t.Helper()
	if _, err := engine.Spawn(name, a); err != nil {
		t.Fatalf("spawn %s: %v", name, err)
	}
}
