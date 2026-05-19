package call

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/presence"
	"qim/internal/eventbus"

	"go.uber.org/zap"
)

const busyQueryTimeout = 3 * time.Second

type CallManagerActor struct {
	engine    *actor.Engine
	events    eventbus.Bus
	callStore CallStore

	busy          map[uint64]string
	callActorRefs map[string]*actor.ActorRef
}

func NewCallManagerActor(engine *actor.Engine, events eventbus.Bus, callStore CallStore) *CallManagerActor {
	return &CallManagerActor{
		engine:        engine,
		events:        events,
		callStore:     callStore,
		busy:          make(map[uint64]string),
		callActorRefs: make(map[string]*actor.ActorRef),
	}
}

func (a *CallManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case InitiateCallCmd:
		a.handleInitiate(ctx, msg)
	case AcceptCallCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case RejectCallCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case CancelCallCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case EndCallCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case ForwardOfferCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case ForwardAnswerCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case ForwardIceCmd:
		a.forwardToCallActor(ctx, msg.CallID, msg)
	case GetCallByUserQuery:
		a.handleGetCallByUser(ctx, msg)
	case actor.Terminated:
		a.handleTerminated(msg)
	}
}

func (a *CallManagerActor) handleInitiate(ctx actor.Context, msg InitiateCallCmd) {
	if msg.CallType != CallTypeVoice && msg.CallType != CallTypeVideo {
		ctx.Reply(Result{Err: ErrInvalidType})
		return
	}
	if msg.CallerUID == msg.CalleeUID {
		ctx.Reply(Result{Err: ErrSelfCall})
		return
	}
	if _, busy := a.busy[msg.CallerUID]; busy {
		ctx.Reply(Result{Err: ErrSelfBusy})
		return
	}
	if !a.isOnline(ctx, msg.CalleeUID) {
		ctx.Reply(Result{Err: ErrOffline})
		return
	}
	if _, busy := a.busy[msg.CalleeUID]; busy {
		ctx.Reply(Result{Err: ErrBusy})
		return
	}

	callID := GenerateCallID()
	callerGW := a.gatewayNameFromUID(ctx, msg.CallerUID)

	callerInfo := CallerInfo{
		Nickname: "",
		Avatar:   "",
	}
	if uinfo, err := a.lookupUserInfo(ctx, msg.CallerUID); err == nil {
		callerInfo = uinfo
	}

	callActor := NewCallActor(a.engine, a.events, a.callStore, StartCallCmd{
		CallID:    callID,
		CallerUID: msg.CallerUID,
		CalleeUID: msg.CalleeUID,
		CallType:  msg.CallType,
		CallerGW:  callerGW,
		CallerInfo: callerInfo,
	})
	ref, err := a.engine.Spawn(fmt.Sprintf("call:%s", callID), callActor)
	if err != nil {
		ctx.Reply(Result{Err: fmt.Errorf("failed to spawn call actor: %w", err)})
		return
	}

	a.busy[msg.CallerUID] = callID
	a.busy[msg.CalleeUID] = callID
	a.callActorRefs[callID] = ref
	ctx.Watch(ref)

	ctx.Reply(Result{Data: InitiateResult{CallID: callID}})
}

func (a *CallManagerActor) forwardToCallActor(ctx actor.Context, callID string, cmd any) {
	ref, ok := a.callActorRefs[callID]
	if !ok {
		ctx.Reply(Result{Err: ErrNotFound})
		return
	}
	raw, err := ref.Ask(cmd, busyQueryTimeout)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	r, ok := raw.(Result)
	if !ok {
		ctx.Reply(Result{Err: fmt.Errorf("unexpected response")})
		return
	}
	ctx.Reply(r)
}

func (a *CallManagerActor) handleGetCallByUser(ctx actor.Context, msg GetCallByUserQuery) {
	callID, busy := a.busy[msg.UID]
	if !busy {
		ctx.Reply(Result{Data: nil})
		return
	}
	ctx.Reply(Result{Data: CallInfo{CallID: callID}})
}

func (a *CallManagerActor) handleTerminated(msg actor.Terminated) {
	refName := msg.Who.Name()
	for callID, ref := range a.callActorRefs {
		if ref.Name() == refName {
			delete(a.callActorRefs, callID)
			a.cleanupCall(callID)
			return
		}
	}
}

func (a *CallManagerActor) cleanupCall(callID string) {
	for uid, cid := range a.busy {
		if cid == callID {
			delete(a.busy, uid)
		}
	}
}

func (a *CallManagerActor) isOnline(ctx actor.Context, uid uint64) bool {
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return false
	}
	raw, err := presenceRef.Ask(presence.BatchOnlineQuery{UIDs: []uint64{uid}}, busyQueryTimeout)
	if err != nil {
		zap.L().Warn("check online failed", zap.Uint64("uid", uid), zap.Error(err))
		return false
	}
	result, ok := raw.(presence.BatchOnlineResult)
	if !ok {
		return false
	}
	return result.OnlineMap[uid]
}

func (a *CallManagerActor) gatewayNameFromUID(ctx actor.Context, uid uint64) string {
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return ""
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: uid}, busyQueryTimeout)
	if err != nil {
		return ""
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok || len(result.Gateways) == 0 {
		return ""
	}
	return result.Gateways[0].Name()
}

func (a *CallManagerActor) lookupUserInfo(ctx actor.Context, uid uint64) (CallerInfo, error) {
	return CallerInfo{Nickname: fmt.Sprintf("用户%d", uid), Avatar: ""}, nil
}
