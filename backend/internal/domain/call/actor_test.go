package call

import (
	"sync"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/eventbus"
)

type testCallStore struct {
	mu      sync.Mutex
	created []*Call
}

func (s *testCallStore) Create(call *Call) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.created = append(s.created, call)
	return nil
}

func (s *testCallStore) GetByID(id uint64) (*Call, error) { return nil, nil }
func (s *testCallStore) ListByUser(uid uint64, offset, limit int) ([]Call, error) {
	return nil, nil
}

func (s *testCallStore) lastCall() *Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.created) == 0 {
		return nil
	}
	return s.created[len(s.created)-1]
}

type testEventBus struct {
	mu     sync.Mutex
	events []eventbus.Event
}

func (b *testEventBus) Publish(event eventbus.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

func (b *testEventBus) Subscribe(eventName string, subscriber *actor.ActorRef) error { return nil }
func (b *testEventBus) Unsubscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}

func (b *testEventBus) count(name string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, e := range b.events {
		if e.Name() == name {
			n++
		}
	}
	return n
}

func (b *testEventBus) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = nil
}

type actorFunc func(ctx actor.Context)

func (f actorFunc) Receive(ctx actor.Context) { f(ctx) }

func spawnCallActor(t *testing.T, engine *actor.Engine, bus eventbus.Bus, store CallStore, cmd StartCallCmd) *actor.ActorRef {
	t.Helper()
	callActor := NewCallActor(engine, bus, store, cmd)
	ref, err := engine.Spawn("call:"+cmd.CallID, callActor)
	if err != nil {
		t.Fatalf("spawn call actor: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	return ref
}

func TestCallActor_RingingToConnectedToEnded_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_test_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	if bus.count(EventCallIncoming) != 1 {
		t.Fatalf("expected 1 incoming event, got %d", bus.count(EventCallIncoming))
	}
	if bus.count(EventCallCalling) != 1 {
		t.Fatalf("expected 1 calling event, got %d", bus.count(EventCallCalling))
	}

	raw, err := ref.Ask(AcceptCallCmd{CallID: "call_test_1", UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("accept ask: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("accept error: %v", r.Err)
	}
	if bus.count(EventCallAccepted) != 1 {
		t.Fatalf("expected 1 accepted event, got %d", bus.count(EventCallAccepted))
	}

	_, err = ref.Ask(EndCallCmd{CallID: "call_test_1", UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("end ask: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record written")
	}
	if rec.Status != CallStatusCompleted {
		t.Fatalf("expected completed, got %d", rec.Status)
	}
	if rec.Duration < 0 {
		t.Fatalf("expected non-negative duration, got %d", rec.Duration)
	}
	if bus.count(EventCallEnded) != 1 {
		t.Fatalf("expected 1 ended event, got %d", bus.count(EventCallEnded))
	}
}

func TestCallActor_Reject_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_reject_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	raw, err := ref.Ask(RejectCallCmd{CallID: "call_reject_1", UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("reject ask: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("reject error: %v", r.Err)
	}
	time.Sleep(50 * time.Millisecond)

	if bus.count(EventCallRejected) != 1 {
		t.Fatalf("expected 1 rejected event, got %d", bus.count(EventCallRejected))
	}
	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record written")
	}
	if rec.Status != CallStatusMissed {
		t.Fatalf("expected missed, got %d", rec.Status)
	}
	if rec.EndReason != "rejected" {
		t.Fatalf("expected rejected, got %s", rec.EndReason)
	}
}

func TestCallActor_Cancel_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_cancel_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	raw, err := ref.Ask(CancelCallCmd{CallID: "call_cancel_1", UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("cancel ask: %v", err)
	}
	r := raw.(Result)
	if r.Err != nil {
		t.Fatalf("cancel error: %v", r.Err)
	}
	time.Sleep(50 * time.Millisecond)

	if bus.count(EventCallCancelled) != 1 {
		t.Fatalf("expected 1 cancelled event, got %d", bus.count(EventCallCancelled))
	}
	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record written")
	}
	if rec.EndReason != "cancelled" {
		t.Fatalf("expected cancelled, got %s", rec.EndReason)
	}
}

func TestCallActor_Timeout_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_timeout_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	ref.Tell(TimeoutCallCmd{})
	time.Sleep(50 * time.Millisecond)

	if bus.count(EventCallTimeout) != 1 {
		t.Fatalf("expected 1 timeout event, got %d", bus.count(EventCallTimeout))
	}
	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record written")
	}
	if rec.EndReason != "timeout" {
		t.Fatalf("expected timeout, got %s", rec.EndReason)
	}
	if rec.Status != CallStatusMissed {
		t.Fatalf("expected missed, got %d", rec.Status)
	}
}

func TestCallActor_DoubleAccept_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_double_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	raw, err := ref.Ask(AcceptCallCmd{CallID: "call_double_1", UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("first accept error: %v", raw.(Result).Err)
	}

	raw, err = ref.Ask(AcceptCallCmd{CallID: "call_double_1", UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("second accept: %v", err)
	}
	if raw.(Result).Err != ErrAlreadyAnswered {
		t.Fatalf("expected already answered, got %v", raw.(Result).Err)
	}
}

func TestCallActor_NonParticipantForbidden_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_forbid_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	tests := []struct {
		name string
		cmd  any
	}{
		{"accept by non-callee", AcceptCallCmd{CallID: "call_forbid_1", UID: 999}},
		{"reject by non-callee", RejectCallCmd{CallID: "call_forbid_1", UID: 999}},
		{"cancel by non-caller", CancelCallCmd{CallID: "call_forbid_1", UID: 999}},
		{"end by outsider", EndCallCmd{CallID: "call_forbid_1", UID: 999}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := ref.Ask(tt.cmd, time.Second)
			if err != nil {
				t.Fatalf("%s ask: %v", tt.name, err)
			}
			if raw.(Result).Err != ErrForbidden {
				t.Fatalf("%s expected forbidden, got %v", tt.name, raw.(Result).Err)
			}
		})
	}
}

func TestCallActor_InvalidState_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_state_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	ref.Ask(AcceptCallCmd{CallID: "call_state_1", UID: 200}, time.Second)

	raw, err := ref.Ask(RejectCallCmd{CallID: "call_state_1", UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("reject after accept: %v", err)
	}
	if raw.(Result).Err != ErrInvalidState {
		t.Fatalf("expected invalid state, got %v", raw.(Result).Err)
	}

	raw, err = ref.Ask(CancelCallCmd{CallID: "call_state_1", UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("cancel after accept: %v", err)
	}
	if raw.(Result).Err != ErrInvalidState {
		t.Fatalf("expected invalid state, got %v", raw.(Result).Err)
	}
}

func TestCallActor_CallerEndsCall_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_caller_end",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVideo,
		CallerGW:  "gw:100",
	})

	ref.Ask(AcceptCallCmd{CallID: "call_caller_end", UID: 200}, time.Second)

	raw, err := ref.Ask(EndCallCmd{CallID: "call_caller_end", UID: 100}, time.Second)
	if err != nil {
		t.Fatalf("caller end: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("caller end error: %v", raw.(Result).Err)
	}
	time.Sleep(50 * time.Millisecond)

	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record")
	}
	if rec.EndReason != "hangup" {
		t.Fatalf("expected hangup, got %s", rec.EndReason)
	}
}

func TestCallActor_CalleeEndsCall_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_callee_end",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	ref.Ask(AcceptCallCmd{CallID: "call_callee_end", UID: 200}, time.Second)

	raw, err := ref.Ask(EndCallCmd{CallID: "call_callee_end", UID: 200}, time.Second)
	if err != nil {
		t.Fatalf("callee end: %v", err)
	}
	if raw.(Result).Err != nil {
		t.Fatalf("callee end error: %v", raw.(Result).Err)
	}
	time.Sleep(50 * time.Millisecond)

	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record")
	}
	if rec.EndReason != "hangup" {
		t.Fatalf("expected hangup, got %s", rec.EndReason)
	}
}

func TestCallActor_OnStartWithoutStartCallCmd_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	callActor := NewCallActor(engine, bus, store, StartCallCmd{})
	ref, err := engine.Spawn("call:badstart", callActor)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = ref.Ask(EndCallCmd{CallID: "xx", UID: 100}, 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout on dead actor")
	}
}

func TestCallActor_GenerateCallID_BitsUT(t *testing.T) {
	id1 := GenerateCallID()
	id2 := GenerateCallID()
	if id1 == id2 {
		t.Fatalf("call IDs should be unique: %s", id1)
	}
	if len(id1) < 10 {
		t.Fatalf("call ID too short: %s", id1)
	}
}

func TestCallActor_ForwardOfferForbidden_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_fwd_offer",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	raw, err := ref.Ask(ForwardOfferCmd{CallID: "call_fwd_offer", UID: 999, SDP: "test"}, time.Second)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if raw.(Result).Err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", raw.(Result).Err)
	}
}

func TestCallActor_ForwardAnswerForbidden_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_fwd_answer",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	raw, err := ref.Ask(ForwardAnswerCmd{CallID: "call_fwd_answer", UID: 999, SDP: "test"}, time.Second)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if raw.(Result).Err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", raw.(Result).Err)
	}
}

func TestCallActor_ForwardIceForbidden_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_fwd_ice",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	raw, err := ref.Ask(ForwardIceCmd{CallID: "call_fwd_ice", UID: 999, Candidate: "c"}, time.Second)
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if raw.(Result).Err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", raw.(Result).Err)
	}
}

func TestCallActor_ForwardSignalInvalidState_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_fwd_state",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	tests := []struct {
		name string
		cmd  any
	}{
		{"offer invalid state", ForwardOfferCmd{CallID: "call_fwd_state", UID: 100, SDP: "test"}},
		{"answer invalid state", ForwardAnswerCmd{CallID: "call_fwd_state", UID: 200, SDP: "test"}},
		{"ice invalid state", ForwardIceCmd{CallID: "call_fwd_state", UID: 100, Candidate: "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := ref.Ask(tt.cmd, time.Second)
			if err != nil {
				t.Fatalf("%s ask: %v", tt.name, err)
			}
			if raw.(Result).Err != ErrInvalidState {
				t.Fatalf("%s expected invalid state, got %v", tt.name, raw.(Result).Err)
			}
		})
	}
}

func TestCallActor_TerminatedCallerGWDuringRinging_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	gwActor := actorFunc(func(ctx actor.Context) {})
	gwRef, err := engine.Spawn("gw:100", gwActor)
	if err != nil {
		t.Fatalf("spawn gw: %v", err)
	}

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_term_1",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	if err := ref.Tell(actor.Terminated{Who: gwRef}); err != nil {
		t.Fatalf("tell terminated: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if bus.count(EventCallCancelled) != 1 {
		t.Fatalf("expected 1 cancelled event, got %d", bus.count(EventCallCancelled))
	}
	rec := store.lastCall()
	if rec == nil {
		t.Fatal("no call record")
	}
	if rec.EndReason != "disconnect" {
		t.Fatalf("expected disconnect, got %s", rec.EndReason)
	}
}

func TestCallActor_TerminatedWrongGW_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	gwActor := actorFunc(func(ctx actor.Context) {})
	gwRef, err := engine.Spawn("gw:other", gwActor)
	if err != nil {
		t.Fatalf("spawn gw: %v", err)
	}

	ref := spawnCallActor(t, engine, bus, store, StartCallCmd{
		CallID:    "call_term_wrong",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	if err := ref.Tell(actor.Terminated{Who: gwRef}); err != nil {
		t.Fatalf("tell terminated: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if bus.count(EventCallCancelled) != 0 {
		t.Fatal("should not cancel for wrong GW")
	}
}

func TestCallActor_PublishEventNil_BitsUT(t *testing.T) {
	store := &testCallStore{}
	bus := &testEventBus{}
	engine := actor.NewEngine()

	a := NewCallActor(engine, nil, store, StartCallCmd{
		CallID:    "call_nil_ev",
		CallerUID: 100,
		CalleeUID: 200,
		CallType:  CallTypeVoice,
		CallerGW:  "gw:100",
	})

	a.publishEvent(CallAcceptedEvent{CallID: "x", CallerUID: 1, CalleeUID: 2})
	if bus.count(EventCallAccepted) != 0 {
		t.Fatal("should not publish when events is nil")
	}
}
