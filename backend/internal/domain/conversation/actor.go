package conversation

import (
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation/store"
	"qim/internal/eventbus"
)

const idleTimeout = 6 * time.Hour
const realtimePushActorName = "message-push"

type ConversationActor struct {
	convID      uint64
	convType    store.ConvType
	name        string
	avatar      string
	ownerID     uint64
	maxSeq      int64
	members     map[uint64]*MemberState
	memberLimit int
	createdAt   int64
	store       store.Store
	engine      *actor.Engine
	events      eventbus.Bus
	idleTimer   *actor.Timer
}

type MemberState struct {
	UID         uint64
	Role        store.MemberRole
	LastReadSeq int64
	JoinTime    int64
}

func NewConversationActor(convID uint64, s store.Store, engine *actor.Engine, events eventbus.Bus) *ConversationActor {
	return &ConversationActor{
		convID:  convID,
		members: make(map[uint64]*MemberState),
		store:   s,
		engine:  engine,
		events:  events,
	}
}

func (a *ConversationActor) OnStart(ctx actor.Context) {
	conv, err := a.store.GetConversation(a.convID)
	if err != nil {
		ctx.Self().Tell(actor.PoisonPill{})
		return
	}
	a.convType = store.ConvType(conv.Type)
	a.name = conv.Name
	a.avatar = conv.Avatar
	a.ownerID = conv.OwnerID
	a.maxSeq = conv.MaxSeq
	a.memberLimit = conv.MemberLimit
	a.createdAt = conv.CreatedAt

	members, err := a.store.GetMembers(a.convID)
	if err != nil {
		ctx.Self().Tell(actor.PoisonPill{})
		return
	}
	for _, m := range members {
		a.members[m.UserID] = &MemberState{
			UID:         m.UserID,
			Role:        store.MemberRole(m.Role),
			LastReadSeq: m.LastReadSeq,
			JoinTime:    m.JoinTime,
		}
	}
	a.resetIdleTimer(ctx)
}

func (a *ConversationActor) OnStop(ctx actor.Context) {
	if a.idleTimer != nil {
		a.idleTimer.Cancel()
	}
}

func (a *ConversationActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case IdleTimeout:
		ctx.Self().Tell(actor.PoisonPill{})
		return
	case GetConvInfoQuery:
		ctx.Reply(Result{Data: a.toDTO()})
	case SendMessageCmd:
		a.handleSendMessage(ctx, msg)
	case RevokeMessageCmd:
		a.handleRevokeMessage(ctx, msg)
	case TypingCmd:
		a.handleTyping(ctx, msg)
	case UpdateConvInfoCmd:
		a.handleUpdateInfo(ctx, msg)
	case ListMembersQuery:
		a.handleListMembers(ctx)
	case AddMemberCmd:
		a.handleAddMember(ctx, msg)
	case RemoveMemberCmd:
		a.handleRemoveMember(ctx, msg)
	case LeaveConvCmd:
		a.handleLeave(ctx, msg)
	case SetRoleCmd:
		a.handleSetRole(ctx, msg)
	case TransferOwnerCmd:
		a.handleTransferOwner(ctx, msg)
	case DissolveConvCmd:
		a.handleDissolve(ctx, msg)
	}
	a.resetIdleTimer(ctx)
}

func (a *ConversationActor) resetIdleTimer(ctx actor.Context) {
	if a.idleTimer != nil {
		a.idleTimer.Cancel()
	}
	a.idleTimer = ctx.ScheduleAfter(idleTimeout, IdleTimeout{})
}

func (a *ConversationActor) toDTO() ConversationDTO {
	return ConversationDTO{
		ID:          a.convID,
		Type:        a.convType,
		Name:        a.name,
		Avatar:      a.avatar,
		OwnerID:     a.ownerID,
		MemberCount: len(a.members),
		MemberLimit: a.memberLimit,
		MaxSeq:      a.maxSeq,
		CreatedAt:   a.createdAt,
	}
}

func (a *ConversationActor) handleTyping(ctx actor.Context, msg TypingCmd) {
	if _, exists := a.members[msg.UID]; !exists {
		ctx.Reply(Result{Err: ErrNotMember})
		return
	}
	if a.convType == store.ConvTypePrivate {
		a.pushTypingToPeer(msg.UID)
	}
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) pushTypingToPeer(fromUID uint64) {
	if a.engine == nil {
		return
	}
	peerUID := a.privatePeerUID(fromUID)
	if peerUID == 0 {
		return
	}
	pushRef, ok := a.engine.Lookup(realtimePushActorName)
	if !ok {
		return
	}
	_ = pushRef.Tell(TypingPushCmd{
		ConversationID: a.convID,
		FromUID:        fromUID,
		ToUID:          peerUID,
	})
}

func (a *ConversationActor) handleUpdateInfo(ctx actor.Context, msg UpdateConvInfoCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.requireAdmin(msg.OperatorID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	now := time.Now().Unix()
	updates := map[string]any{"updated_at": now}
	if msg.Name != nil {
		updates["name"] = *msg.Name
	}
	if msg.Avatar != nil {
		updates["avatar"] = *msg.Avatar
	}
	if msg.MemberLimit != nil {
		if *msg.MemberLimit < len(a.members) {
			ctx.Reply(Result{Err: ErrMemberLimitReached})
			return
		}
		updates["member_limit"] = *msg.MemberLimit
	}

	if err := a.store.UpdateConversation(a.convID, updates); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if msg.Name != nil {
		a.name = *msg.Name
	}
	if msg.Avatar != nil {
		a.avatar = *msg.Avatar
	}
	if msg.MemberLimit != nil {
		a.memberLimit = *msg.MemberLimit
	}
	ctx.Reply(Result{Data: true})
	a.publishConversationUpdated()
}

func (a *ConversationActor) handleListMembers(ctx actor.Context) {
	members := make([]MemberDTO, 0, len(a.members))
	for _, m := range a.members {
		members = append(members, MemberDTO{
			UID:         m.UID,
			Role:        m.Role,
			LastReadSeq: m.LastReadSeq,
			JoinTime:    m.JoinTime,
		})
	}
	ctx.Reply(Result{Data: members})
}

func (a *ConversationActor) handleAddMember(ctx actor.Context, msg AddMemberCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.requireAdmin(msg.OperatorID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if msg.Role == store.MemberRoleOwner || !isAssignableRole(msg.Role) {
		ctx.Reply(Result{Err: ErrInvalidRole})
		return
	}
	if _, exists := a.members[msg.UID]; exists {
		ctx.Reply(Result{Err: ErrMemberExists})
		return
	}
	if a.memberLimit > 0 && len(a.members) >= a.memberLimit {
		ctx.Reply(Result{Err: ErrMemberLimitReached})
		return
	}

	now := time.Now().Unix()
	member := &store.MemberRecord{
		ConversationID: a.convID,
		UserID:         msg.UID,
		Role:           int8(msg.Role),
		JoinTime:       now,
	}
	if err := a.store.CreateMember(member); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	a.members[msg.UID] = &MemberState{
		UID:      msg.UID,
		Role:     msg.Role,
		JoinTime: now,
	}
	ctx.Reply(Result{Data: true})
	a.publishMemberJoined(msg.UID, msg.Role, msg.OperatorID)
}

func (a *ConversationActor) handleRemoveMember(ctx actor.Context, msg RemoveMemberCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.requireAdmin(msg.OperatorID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if _, exists := a.members[msg.UID]; !exists {
		ctx.Reply(Result{Err: ErrMemberNotFound})
		return
	}
	if msg.UID == a.ownerID {
		ctx.Reply(Result{Err: ErrOwnerRequired})
		return
	}
	if err := a.store.DeleteMember(a.convID, msg.UID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	delete(a.members, msg.UID)
	ctx.Reply(Result{Data: true})
	a.publishMemberKicked(msg.UID, msg.OperatorID)
}

func (a *ConversationActor) handleLeave(ctx actor.Context, msg LeaveConvCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if _, exists := a.members[msg.UID]; !exists {
		ctx.Reply(Result{Err: ErrNotMember})
		return
	}
	if msg.UID == a.ownerID {
		ctx.Reply(Result{Err: ErrOwnerRequired})
		return
	}
	if err := a.store.DeleteMember(a.convID, msg.UID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	delete(a.members, msg.UID)
	ctx.Reply(Result{Data: true})
	a.publishMemberLeft(msg.UID)
}

func (a *ConversationActor) handleSetRole(ctx actor.Context, msg SetRoleCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.requireOwner(msg.OperatorID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if msg.Role == store.MemberRoleOwner || !isAssignableRole(msg.Role) {
		ctx.Reply(Result{Err: ErrInvalidRole})
		return
	}
	m, exists := a.members[msg.UID]
	if !exists {
		ctx.Reply(Result{Err: ErrMemberNotFound})
		return
	}
	if msg.UID == a.ownerID {
		ctx.Reply(Result{Err: ErrInvalidRole})
		return
	}
	if err := a.store.UpdateMember(a.convID, msg.UID, map[string]any{"role": msg.Role}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	m.Role = msg.Role
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleTransferOwner(ctx actor.Context, msg TransferOwnerCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.requireOwner(msg.OperatorID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	newOwner, exists := a.members[msg.NewOwnerID]
	if !exists {
		ctx.Reply(Result{Err: ErrMemberNotFound})
		return
	}
	oldOwner := a.members[a.ownerID]

	if err := a.store.TransferOwner(a.convID, a.ownerID, msg.NewOwnerID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	oldOwner.Role = store.MemberRoleRegular
	newOwner.Role = store.MemberRoleOwner
	oldOwnerID := a.ownerID
	a.ownerID = msg.NewOwnerID
	ctx.Reply(Result{Data: true})
	a.publishOwnerTransferred(oldOwnerID, msg.NewOwnerID)
}

func (a *ConversationActor) handleDissolve(ctx actor.Context, msg DissolveConvCmd) {
	if err := a.requireGroup(); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.requireOwner(msg.OperatorID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	memberUIDs := a.memberUIDs()
	if err := a.store.DissolveConversation(a.convID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
	a.publishGroupDissolved(msg.OperatorID, memberUIDs)
	ctx.Self().Tell(actor.PoisonPill{})
}

func (a *ConversationActor) publishConversationUpdated() {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(ConversationUpdatedEvent{
		ConversationID: a.convID,
		DisplayName:    a.name,
		Avatar:         a.avatar,
		MemberUIDs:     a.memberUIDs(),
	})
}

func (a *ConversationActor) publishMemberJoined(uid uint64, role store.MemberRole, operatorID uint64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(MemberJoinedEvent{
		ConversationID: a.convID,
		UID:            uid,
		Role:           role,
		OperatorID:     operatorID,
		MemberUIDs:     a.memberUIDs(),
	})
}

func (a *ConversationActor) publishMemberLeft(uid uint64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(MemberLeftEvent{
		ConversationID: a.convID,
		UID:            uid,
		MemberUIDs:     append(a.memberUIDs(), uid),
	})
}

func (a *ConversationActor) publishMemberKicked(uid, operatorID uint64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(MemberKickedEvent{
		ConversationID: a.convID,
		UID:            uid,
		OperatorID:     operatorID,
		MemberUIDs:     append(a.memberUIDs(), uid),
	})
}

func (a *ConversationActor) publishOwnerTransferred(oldOwnerID, newOwnerID uint64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(OwnerTransferredEvent{
		ConversationID: a.convID,
		OldOwnerID:     oldOwnerID,
		NewOwnerID:     newOwnerID,
		MemberUIDs:     a.memberUIDs(),
	})
}

func (a *ConversationActor) publishGroupDissolved(operatorID uint64, memberUIDs []uint64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(GroupDissolvedEvent{
		ConversationID: a.convID,
		OperatorID:     operatorID,
		MemberUIDs:     memberUIDs,
	})
}

func (a *ConversationActor) memberUIDs() []uint64 {
	uids := make([]uint64, 0, len(a.members))
	for uid := range a.members {
		uids = append(uids, uid)
	}
	return uids
}

func (a *ConversationActor) privatePeerUID(uid uint64) uint64 {
	for memberUID := range a.members {
		if memberUID != uid {
			return memberUID
		}
	}
	return 0
}

func (a *ConversationActor) requireAdmin(uid uint64) error {
	member, ok := a.members[uid]
	if !ok {
		return ErrNotMember
	}
	if member.Role != store.MemberRoleOwner && member.Role != store.MemberRoleAdmin {
		return ErrAdminRequired
	}
	return nil
}

func (a *ConversationActor) requireOwner(uid uint64) error {
	if uid == 0 || uid != a.ownerID {
		return ErrOwnerRequired
	}
	return nil
}

func (a *ConversationActor) requireGroup() error {
	if a.convType != store.ConvTypeGroup {
		return ErrGroupRequired
	}
	return nil
}

func isAssignableRole(role store.MemberRole) bool {
	return role == store.MemberRoleRegular || role == store.MemberRoleAdmin
}
