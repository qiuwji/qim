package call

import (
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/domain/presence"
	"qim/internal/eventbus"
)

func setupCallManager(t *testing.T) (*actor.Engine, eventbus.Bus, *testCallStore, *actor.ActorRef, *actor.ActorRef) {
	t.Helper()
	engine := actor.NewEngine()
	bus := &testEventBus{}
	store := &testCallStore{}

	presenceActor := presence.NewPresenceActor(engine, nil)
	presenceRef, err := engine.Spawn("presence", presenceActor)
	if err != nil {
		t.Fatalf("spawn presence: %v", err)
	}

	var convStore conversation.Store
	mgrRef, err := engine.Spawn("call-manager", NewCallManagerActor(engine, bus, store, convStore))
	if err != nil {
		t.Fatalf("spawn call-manager: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	return engine, bus, store, mgrRef, presenceRef
}

func registerGateway(t *testing.T, engine *actor.Engine, presenceRef *actor.ActorRef, uid uint64, gwName string) *actor.ActorRef {
	t.Helper()
	gwActor := actorFunc(func(ctx actor.Context) {})
	gwRef, err := engine.Spawn(gwName, gwActor)
	if err != nil {
		t.Fatalf("spawn gw: %v", err)
	}
	if err := presenceRef.Tell(presence.UserConnected{UID: uid, Gateway: gwRef}); err != nil {
		t.Fatalf("register gw: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	return gwRef
}

func TestCallManager_InitiateSuccess_BitsUT(t *testing.T) {
	engine, bus, store, mgrRef, presenceRef := setupCallManager(t)
	_ = store

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate ask: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("initiate error: %v", r.Err)
	}
	initResult := r.Data.(InitiateResult)
	if initResult.CallID == "" {
		t.Fatal("expected call_id in result")
	}

	time.Sleep(50 * time.Millisecond)
	tb := bus.(*testEventBus)
	if tb.count(EventCallIncoming) != 1 {
		t.Fatalf("expected 1 incoming event, got %d", tb.count(EventCallIncoming))
	}
}

func TestCallManager_SelfBusy_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")
	registerGateway(t, engine, presenceRef, 300, "gw:300")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("first initiate: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("first initiate error: %v", raw.(Result).Err)
	}

	raw, err = mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 300,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("second initiate: %v", err)
	}
	if raw.(Result).Err != ErrSelfBusy {
		t.Fatalf("expected self_busy, got %v", raw.(Result).Err)
	}
}

func TestCallManager_CalleeBusy_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")
	registerGateway(t, engine, presenceRef, 300, "gw:300")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("first initiate: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("first initiate error: %v", raw.(Result).Err)
	}

	raw, err = mgrRef.Ask(InitiateCallCmd{
		CallerUID: 300,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("second initiate: %v", err)
	}
	if raw.(Result).Err != ErrBusy {
		t.Fatalf("expected busy, got %v", raw.(Result).Err)
	}
}

func TestCallManager_Offline_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 999,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if raw.(Result).Err != ErrOffline {
		t.Fatalf("expected offline, got %v", raw.(Result).Err)
	}
}

func TestCallManager_SelfCall_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 100,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if raw.(Result).Err != ErrSelfCall {
		t.Fatalf("expected self_call, got %v", raw.(Result).Err)
	}
}

func TestCallManager_InvalidType_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  99,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if raw.(Result).Err != ErrInvalidType {
		t.Fatalf("expected invalid_type, got %v", raw.(Result).Err)
	}
}

func TestCallManager_TerminatedCleanupBusy_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("first initiate: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("first initiate error: %v", r.Err)
	}
	initResult := r.Data.(InitiateResult)

	callRef, ok := engine.Lookup("call:" + initResult.CallID)
	if !ok {
		t.Fatal("call actor not found")
	}

	if err := callRef.Tell(EndCallCmd{CallID: initResult.CallID, UID: 100}); err != nil {
		t.Fatalf("end call: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	raw, err = mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("second initiate: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("second initiate should succeed after cleanup, got %v", raw.(Result).Err)
	}
}

func TestCallManager_NotFound_BitsUT(t *testing.T) {
	_, _, _, mgrRef, _ := setupCallManager(t)

	raw, err := mgrRef.Ask(AcceptCallCmd{CallID: "nonexistent", UID: 200}, 5*time.Second)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if raw.(Result).Err != ErrNotFound {
		t.Fatalf("expected not_found, got %v", raw.(Result).Err)
	}
}

func TestCallManager_GetCallByUser_NoCall_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")

	raw, err := mgrRef.Ask(GetCallByUserQuery{UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("error: %v", r.Err)
	}
	if r.Data != nil {
		t.Fatal("expected nil data for user without call")
	}
}

func TestCallManager_GetCallByUser_WithCall_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("initiate error: %v", raw.(Result).Err)
	}

	raw, err = mgrRef.Ask(GetCallByUserQuery{UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("error: %v", r.Err)
	}
	if r.Data == nil {
		t.Fatal("expected call info")
	}
	info := r.Data.(CallInfo)
	if info.CallID == "" {
		t.Fatal("expected call_id")
	}
}

func TestCallManager_AcceptRejectCancelEnd_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("initiate error: %v", r.Err)
	}
	callID := r.Data.(InitiateResult).CallID

	raw, err = mgrRef.Ask(AcceptCallCmd{CallID: callID, UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("accept error: %v", raw.(Result).Err)
	}

	raw, err = mgrRef.Ask(EndCallCmd{CallID: callID, UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("end error: %v", raw.(Result).Err)
	}
}

func TestCallManager_ForwardOffer_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("initiate error: %v", r.Err)
	}
	callID := r.Data.(InitiateResult).CallID

	mgrRef.Ask(AcceptCallCmd{CallID: callID, UID: 200}, time.Second)

	raw, err = mgrRef.Ask(ForwardOfferCmd{CallID: callID, UID: 100, SDP: "test_sdp"}, time.Second)
	if err != nil {
		t.Fatalf("forward offer: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("forward offer error: %v", raw.(Result).Err)
	}
}

func TestCallManager_ForwardAnswer_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("initiate error: %v", r.Err)
	}
	callID := r.Data.(InitiateResult).CallID

	mgrRef.Ask(AcceptCallCmd{CallID: callID, UID: 200}, time.Second)

	raw, err = mgrRef.Ask(ForwardAnswerCmd{CallID: callID, UID: 200, SDP: "test_sdp"}, time.Second)
	if err != nil {
		t.Fatalf("forward answer: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("forward answer error: %v", raw.(Result).Err)
	}
}

func TestCallManager_ForwardIce_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("initiate error: %v", r.Err)
	}
	callID := r.Data.(InitiateResult).CallID

	mgrRef.Ask(AcceptCallCmd{CallID: callID, UID: 200}, time.Second)

	raw, err = mgrRef.Ask(ForwardIceCmd{CallID: callID, UID: 100, Candidate: "cand"}, time.Second)
	if err != nil {
		t.Fatalf("forward ice: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("forward ice error: %v", raw.(Result).Err)
	}
}

func TestCallManager_UnknownMessage_BitsUT(t *testing.T) {
	_, _, _, mgrRef, _ := setupCallManager(t)

	if err := mgrRef.Tell("unknown_message"); err != nil {
		t.Fatalf("tell unknown: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
}

func TestCallManager_LookupUserInfo_BitsUT(t *testing.T) {
	engine, _, _, mgrRef, presenceRef := setupCallManager(t)

	registerGateway(t, engine, presenceRef, 100, "gw:100")
	registerGateway(t, engine, presenceRef, 200, "gw:200")

	raw, err := mgrRef.Ask(InitiateCallCmd{
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVideo,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("initiate error: %v", raw.(Result).Err)
	}
	if raw.(Result).Data.(InitiateResult).CallID == "" {
		t.Fatal("expected call_id")
	}
}
