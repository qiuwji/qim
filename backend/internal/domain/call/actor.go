package call

import (
	"fmt"
	"math/rand"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/presence"
	"qim/internal/eventbus"
	"qim/internal/pkg/pushtype"

	"go.uber.org/zap"
)

const callTimeout = 30 * time.Second
const presenceAskTimeout = time.Second

type callActor struct {
	callID    string
	callerUID uint64
	calleeUID uint64
	callType  int8
	status    string

	callerGW    string
	calleeGW    string
	callerGWRef *actor.ActorRef
	calleeGWRef *actor.ActorRef
	calleeGWRefs []*actor.ActorRef

	callerInfo CallerInfo

	startedAt int64
	endedAt   int64
	endReason string
	answered  bool

	timeoutTimer *actor.Timer

	engine     *actor.Engine
	events     eventbus.Bus
	callStore  CallStore

	startCmd StartCallCmd
}

func NewCallActor(engine *actor.Engine, events eventbus.Bus, callStore CallStore, cmd StartCallCmd) *callActor {
	return &callActor{
		engine:    engine,
		events:    events,
		callStore: callStore,
		startCmd:  cmd,
	}
}

func (a *callActor) OnStart(ctx actor.Context) {
	cmd := a.startCmd
	if cmd.CallID == "" {
		zap.L().Error("call actor started without StartCallCmd")
		ctx.Self().Tell(actor.PoisonPill{})
		return
	}

	a.callID = cmd.CallID
	a.callerUID = cmd.CallerUID
	a.calleeUID = cmd.CalleeUID
	a.callType = cmd.CallType
	a.status = CallStatusRinging
	a.callerGW = cmd.CallerGW
	a.callerInfo = cmd.CallerInfo

	a.timeoutTimer = ctx.ScheduleAfter(callTimeout, TimeoutCallCmd{})

	if a.callerGW != "" {
		callerRef := a.lookupGateway(ctx, a.callerUID, a.callerGW)
		if callerRef != nil {
			a.callerGWRef = callerRef
			ctx.Watch(callerRef)
		}
	}

	calleeRefs := a.lookupAllGateways(ctx, a.calleeUID)
	for _, ref := range calleeRefs {
		ctx.Watch(ref)
	}
	a.calleeGWRefs = calleeRefs

	a.publishEvent(CallIncomingEvent{
		CallID:       a.callID,
		CallerUID:    a.callerUID,
		CalleeUID:    a.calleeUID,
		CallType:     a.callType,
		CallerName:   a.callerInfo.Nickname,
		CallerAvatar: a.callerInfo.Avatar,
	})

	a.publishEvent(CallCallingEvent{
		CallID:    a.callID,
		CallerUID: a.callerUID,
		CalleeUID: a.calleeUID,
	})
}

func (a *callActor) OnStop(ctx actor.Context) {
	if a.timeoutTimer != nil {
		a.timeoutTimer.Cancel()
	}

	now := time.Now().Unix()
	a.endedAt = now

	callStatus := CallStatusMissed
	duration := int64(0)
	if a.startedAt > 0 {
		callStatus = CallStatusCompleted
		duration = now - a.startedAt
	}

	if a.endReason == "" {
		a.endReason = "hangup"
	}

	record := &Call{
		CallerUID: a.callerUID,
		CalleeUID: a.calleeUID,
		CallType:  a.callType,
		Status:    callStatus,
		StartedAt: a.startedAt,
		EndedAt:   now,
		Duration:  duration,
		EndReason: a.endReason,
		CreatedAt: now,
	}
	if err := a.callStore.Create(record); err != nil {
		zap.L().Error("failed to save call record", zap.String("call_id", a.callID), zap.Error(err))
	}

	switch a.endReason {
	case "timeout":
		a.publishEvent(CallTimeoutEvent{
			CallID:    a.callID,
			CallerUID: a.callerUID,
			CalleeUID: a.calleeUID,
		})
	default:
		a.publishEvent(CallEndedEvent{
			CallID:    a.callID,
			CallerUID: a.callerUID,
			CalleeUID: a.calleeUID,
			StartedAt: a.startedAt,
			Duration:  duration,
			EndReason: a.endReason,
		})
	}
}

func (a *callActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case AcceptCallCmd:
		a.handleAccept(ctx, msg)
	case RejectCallCmd:
		a.handleReject(ctx, msg)
	case CancelCallCmd:
		a.handleCancel(ctx, msg)
	case EndCallCmd:
		a.handleEnd(ctx, msg)
	case TimeoutCallCmd:
		a.handleTimeout(ctx)
	case ForwardOfferCmd:
		a.handleForwardOffer(ctx, msg)
	case ForwardAnswerCmd:
		a.handleForwardAnswer(ctx, msg)
	case ForwardIceCmd:
		a.handleForwardIce(ctx, msg)
	case actor.Terminated:
		a.handleTerminated(ctx, msg)
	}
}

func (a *callActor) handleAccept(ctx actor.Context, msg AcceptCallCmd) {
	if msg.UID != a.calleeUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status != CallStatusRinging {
		ctx.Reply(Result{Err: ErrAlreadyAnswered})
		return
	}
	if a.answered {
		ctx.Reply(Result{Err: ErrAlreadyAnswered})
		return
	}

	a.answered = true
	a.status = CallStatusConnected
	a.startedAt = time.Now().Unix()

	gwName := msg.GatewayName
	if gwName == "" {
		gwName = a.firstGatewayName(ctx, a.calleeUID)
	}
	a.calleeGW = gwName

	for _, ref := range a.calleeGWRefs {
		ctx.Unwatch(ref)
	}
	a.calleeGWRefs = nil

	calleeRef := a.lookupGateway(ctx, a.calleeUID, gwName)
	if calleeRef != nil {
		a.calleeGWRef = calleeRef
		ctx.Watch(calleeRef)
	}

	if a.timeoutTimer != nil {
		a.timeoutTimer.Cancel()
		a.timeoutTimer = nil
	}

	ctx.Reply(Result{Data: true})

	a.publishEvent(CallAcceptedEvent{
		CallID:    a.callID,
		CallerUID: a.callerUID,
		CalleeUID: a.calleeUID,
		StartedAt: a.startedAt,
	})

	a.publishEvent(CallAnsweredElsewhereEvent{
		CallID:    a.callID,
		CallerUID: a.callerUID,
		CalleeUID: a.calleeUID,
		AcceptGW:  gwName,
	})
}

func (a *callActor) handleReject(ctx actor.Context, msg RejectCallCmd) {
	if msg.UID != a.calleeUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status != CallStatusRinging {
		ctx.Reply(Result{Err: ErrInvalidState})
		return
	}

	a.status = CallStatusEnded
	a.endReason = "rejected"
	ctx.Reply(Result{Data: true})

	a.publishEvent(CallRejectedEvent{
		CallID:    a.callID,
		CallerUID: a.callerUID,
		CalleeUID: a.calleeUID,
	})

	ctx.Self().Tell(actor.PoisonPill{})
}

func (a *callActor) handleCancel(ctx actor.Context, msg CancelCallCmd) {
	if msg.UID != a.callerUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status != CallStatusRinging {
		ctx.Reply(Result{Err: ErrInvalidState})
		return
	}

	a.status = CallStatusEnded
	a.endReason = "cancelled"
	ctx.Reply(Result{Data: true})

	a.publishEvent(CallCancelledEvent{
		CallID:    a.callID,
		CallerUID: a.callerUID,
		CalleeUID: a.calleeUID,
	})

	ctx.Self().Tell(actor.PoisonPill{})
}

func (a *callActor) handleEnd(ctx actor.Context, msg EndCallCmd) {
	if msg.UID != a.callerUID && msg.UID != a.calleeUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status == CallStatusEnded {
		ctx.Reply(Result{Err: ErrInvalidState})
		return
	}

	a.status = CallStatusEnded
	a.endReason = "hangup"
	ctx.Reply(Result{Data: true})

	ctx.Self().Tell(actor.PoisonPill{})
}

func (a *callActor) handleTimeout(ctx actor.Context) {
	if a.status != CallStatusRinging {
		return
	}

	a.status = CallStatusEnded
	a.endReason = "timeout"

	ctx.Self().Tell(actor.PoisonPill{})
}

func (a *callActor) handleForwardOffer(ctx actor.Context, msg ForwardOfferCmd) {
	if msg.UID != a.callerUID && msg.UID != a.calleeUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status != CallStatusConnected {
		ctx.Reply(Result{Err: ErrInvalidState})
		return
	}
	ctx.Reply(Result{Data: true})
	a.pushSignalToGW(a.calleeGWRef, a.calleeUID, a.calleeGW, "offer", map[string]any{
		"call_id": a.callID,
		"sdp":     msg.SDP,
	})
}

func (a *callActor) handleForwardAnswer(ctx actor.Context, msg ForwardAnswerCmd) {
	if msg.UID != a.callerUID && msg.UID != a.calleeUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status != CallStatusConnected {
		ctx.Reply(Result{Err: ErrInvalidState})
		return
	}
	ctx.Reply(Result{Data: true})
	a.pushSignalToGW(a.callerGWRef, a.callerUID, a.callerGW, "answer", map[string]any{
		"call_id": a.callID,
		"sdp":     msg.SDP,
	})
}

func (a *callActor) handleForwardIce(ctx actor.Context, msg ForwardIceCmd) {
	if msg.UID != a.callerUID && msg.UID != a.calleeUID {
		ctx.Reply(Result{Err: ErrForbidden})
		return
	}
	if a.status != CallStatusConnected {
		ctx.Reply(Result{Err: ErrInvalidState})
		return
	}

	var targetRef *actor.ActorRef
	var targetUID uint64
	var targetGW string

	if msg.UID == a.callerUID {
		targetRef = a.calleeGWRef
		targetUID = a.calleeUID
		targetGW = a.calleeGW
	} else {
		targetRef = a.callerGWRef
		targetUID = a.callerUID
		targetGW = a.callerGW
	}

	ctx.Reply(Result{Data: true})
	a.pushSignalToGW(targetRef, targetUID, targetGW, "ice", map[string]any{
		"call_id":          a.callID,
		"candidate":        msg.Candidate,
		"sdp_mid":          msg.SDPMid,
		"sdp_m_line_index": msg.SDPMLineIndex,
	})
}

func (a *callActor) handleTerminated(ctx actor.Context, msg actor.Terminated) {
	refName := msg.Who.Name()

	switch {
	case a.status == CallStatusRinging && refName == a.callerGW:
		a.status = CallStatusEnded
		a.endReason = "disconnect"
		a.publishEvent(CallCancelledEvent{
			CallID:    a.callID,
			CallerUID: a.callerUID,
			CalleeUID: a.calleeUID,
		})
		ctx.Self().Tell(actor.PoisonPill{})

	case a.status == CallStatusRinging:
		if a.calleeOnlineCount(ctx) == 0 {
			a.status = CallStatusEnded
			a.endReason = "disconnect"
			a.publishEvent(CallCancelledEvent{
				CallID:    a.callID,
				CallerUID: a.callerUID,
				CalleeUID: a.calleeUID,
			})
			ctx.Self().Tell(actor.PoisonPill{})
		}

	case a.status == CallStatusConnected && (refName == a.callerGW || refName == a.calleeGW):
		a.status = CallStatusEnded
		a.endReason = "disconnect"
		ctx.Self().Tell(actor.PoisonPill{})
	}
}

func (a *callActor) lookupGateway(ctx actor.Context, uid uint64, gwName string) *actor.ActorRef {
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return nil
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		zap.L().Warn("lookup gateway failed", zap.Uint64("uid", uid), zap.String("gw", gwName), zap.Error(err))
		return nil
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok {
		return nil
	}
	for _, gw := range result.Gateways {
		if gw.Name() == gwName {
			return gw
		}
	}
	return nil
}

func (a *callActor) firstGatewayName(ctx actor.Context, uid uint64) string {
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return ""
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		return ""
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok || len(result.Gateways) == 0 {
		return ""
	}
	return result.Gateways[0].Name()
}

func (a *callActor) lookupAllGateways(ctx actor.Context, uid uint64) []*actor.ActorRef {
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return nil
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		return nil
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok {
		return nil
	}
	return result.Gateways
}

func (a *callActor) calleeOnlineCount(ctx actor.Context) int {
	gateways := a.lookupAllGateways(ctx, a.calleeUID)
	if gateways == nil {
		return 1
	}
	return len(gateways)
}

func (a *callActor) pushToUser(ctx actor.Context, uid uint64, pushType, action string, data any) {
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		zap.L().Warn("push to user failed", zap.Uint64("uid", uid), zap.Error(err))
		return
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok {
		return
	}
	for _, gw := range result.Gateways {
		_ = gw.Tell(pushtype.PushCmd{Type: pushType, Action: action, Data: data})
	}
}

func (a *callActor) pushSignalToGW(cachedRef *actor.ActorRef, uid uint64, gwName, action string, data map[string]any) {
	if cachedRef != nil {
		if err := cachedRef.Tell(pushtype.PushCmd{Type: "call", Action: action, Data: data}); err == nil {
			return
		}
	}
	if a.engine == nil {
		return
	}
	presenceRef, ok := a.engine.Lookup("presence")
	if !ok {
		return
	}
	raw, err := presenceRef.Ask(presence.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		return
	}
	result, ok := raw.(presence.GatewaysResult)
	if !ok {
		return
	}
	for _, gw := range result.Gateways {
		if gw.Name() == gwName {
			_ = gw.Tell(pushtype.PushCmd{Type: "call", Action: action, Data: data})
			return
		}
	}
}

func (a *callActor) publishEvent(event eventbus.Event) {
	if a.events == nil {
		return
	}
	if err := a.events.Publish(event); err != nil {
		zap.L().Warn("failed to publish call event", zap.String("call_id", a.callID), zap.String("event", event.Name()), zap.Error(err))
	}
}

func randomID(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenerateCallID() string {
	return fmt.Sprintf("call_%d_%s", time.Now().UnixMilli(), randomID(8))
}
