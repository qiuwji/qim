package service

import (
	"strings"
	"testing"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	"qim/internal/domain/message"
	"qim/internal/domain/user"
)

func TestServices_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawn(t, engine, "user-manager", resultActor{result: user.Result{Data: "user"}})
	mustSpawn(t, engine, "conv-manager", resultActor{result: conversation.Result{Data: "conv-manager"}})
	mustSpawn(t, engine, "friend-manager", resultActor{result: friend.Result{Data: "friend"}})
	mustSpawn(t, engine, "msg-store", resultActor{result: message.Result{Data: "message"}})

	t.Run("user service manager 和 session", func(t *testing.T) {
		svc := NewUserService(engine, func(uid uint64) actor.Actor {
			return resultActor{result: user.Result{Data: uid}}
		})
		if got, err := svc.AskManager("cmd"); err != nil || got.Data != "user" {
			t.Fatalf("AskManager got=%+v err=%v", got, err)
		}
		if got, err := svc.AskSession(100, "cmd"); err != nil || got.Data != uint64(100) {
			t.Fatalf("AskSession got=%+v err=%v", got, err)
		}
		if err := svc.TellSession(100, "cmd"); err != nil {
			t.Fatalf("TellSession error: %v", err)
		}
	})

	t.Run("conversation service manager 和会话 actor", func(t *testing.T) {
		svc := NewConvService(engine, func(convID uint64) actor.Actor {
			return resultActor{result: conversation.Result{Data: convID}}
		})
		if got, err := svc.AskManager("cmd"); err != nil || got.Data != "conv-manager" {
			t.Fatalf("AskManager got=%+v err=%v", got, err)
		}
		if got, err := svc.AskConv(200, "cmd"); err != nil || got.Data != uint64(200) {
			t.Fatalf("AskConv got=%+v err=%v", got, err)
		}
		if err := svc.TellConv(200, "cmd"); err != nil {
			t.Fatalf("TellConv error: %v", err)
		}
	})

	t.Run("friend 和 msg service", func(t *testing.T) {
		friendSvc := NewFriendService(engine)
		if got, err := friendSvc.Ask("cmd"); err != nil || got.Data != "friend" {
			t.Fatalf("friend Ask got=%+v err=%v", got, err)
		}
		if err := friendSvc.Tell("cmd"); err != nil {
			t.Fatalf("friend Tell error: %v", err)
		}

		msgSvc := NewMsgService(engine)
		if got, err := msgSvc.Ask("cmd"); err != nil || got.Data != "message" {
			t.Fatalf("msg Ask got=%+v err=%v", got, err)
		}
		if err := msgSvc.Tell("cmd"); err != nil {
			t.Fatalf("msg Tell error: %v", err)
		}
	})
}

func TestServicesUnavailable_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	if _, err := NewUserService(engine, nil).AskManager("cmd"); err == nil || !strings.Contains(err.Error(), "user-manager unavailable") {
		t.Fatalf("expected user-manager unavailable, got %v", err)
	}
	if _, err := NewConvService(engine, nil).AskManager("cmd"); err == nil || !strings.Contains(err.Error(), "conv-manager unavailable") {
		t.Fatalf("expected conv-manager unavailable, got %v", err)
	}
	if _, err := NewFriendService(engine).Ask("cmd"); err == nil || !strings.Contains(err.Error(), "friend-manager unavailable") {
		t.Fatalf("expected friend-manager unavailable, got %v", err)
	}
	if _, err := NewMsgService(engine).Ask("cmd"); err == nil || !strings.Contains(err.Error(), "msg-store unavailable") {
		t.Fatalf("expected msg-store unavailable, got %v", err)
	}
}

func TestServicesUnexpectedResult_BitsUT(t *testing.T) {
	engine := actor.NewEngine()
	mustSpawn(t, engine, "user-manager", resultActor{result: "bad"})
	mustSpawn(t, engine, "conv-manager", resultActor{result: "bad"})
	mustSpawn(t, engine, "friend-manager", resultActor{result: "bad"})
	mustSpawn(t, engine, "msg-store", resultActor{result: "bad"})

	if _, err := NewUserService(engine, func(uid uint64) actor.Actor { return resultActor{result: "bad"} }).AskManager("cmd"); err == nil {
		t.Fatalf("user AskManager should reject unexpected result")
	}
	if _, err := NewUserService(engine, func(uid uint64) actor.Actor { return resultActor{result: "bad"} }).AskSession(1, "cmd"); err == nil {
		t.Fatalf("user AskSession should reject unexpected result")
	}
	if _, err := NewConvService(engine, func(convID uint64) actor.Actor { return resultActor{result: "bad"} }).AskManager("cmd"); err == nil {
		t.Fatalf("conv AskManager should reject unexpected result")
	}
	if _, err := NewConvService(engine, func(convID uint64) actor.Actor { return resultActor{result: "bad"} }).AskConv(1, "cmd"); err == nil {
		t.Fatalf("conv AskConv should reject unexpected result")
	}
	if _, err := NewFriendService(engine).Ask("cmd"); err == nil {
		t.Fatalf("friend Ask should reject unexpected result")
	}
	if _, err := NewMsgService(engine).Ask("cmd"); err == nil {
		t.Fatalf("msg Ask should reject unexpected result")
	}
}

type resultActor struct {
	result any
}

func (a resultActor) Receive(ctx actor.Context) {
	ctx.Reply(a.result)
}

func mustSpawn(t *testing.T, engine *actor.Engine, name string, a actor.Actor) {
	t.Helper()
	if _, err := engine.Spawn(name, a); err != nil {
		t.Fatalf("spawn %s: %v", name, err)
	}
}
