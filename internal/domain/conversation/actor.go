package conversation

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

const idleTimeout = 6 * time.Hour

type ConversationActor struct {
	convID      uint64
	convType    ConvType
	name        string
	avatar      string
	ownerID     uint64
	maxSeq      int64
	members     map[uint64]*MemberState
	memberLimit int
	createdAt   int64
	store       dal.ConvStore
	engine      *actor.Engine
	idleTimer   *actor.Timer
}

type MemberState struct {
	UID         uint64
	Role        MemberRole
	LastReadSeq int64
	JoinTime    int64
}

func NewConversationActor(convID uint64, store dal.ConvStore, engine *actor.Engine) *ConversationActor {
	return &ConversationActor{
		convID:  convID,
		members: make(map[uint64]*MemberState),
		store:   store,
		engine:  engine,
	}
}

func (a *ConversationActor) OnStart(ctx actor.Context) {
	conv, err := a.store.GetConversation(a.convID)
	if err != nil {
		ctx.Self().Tell(actor.PoisonPill{})
		return
	}
	a.convType = ConvType(conv.Type)
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
			Role:        MemberRole(m.Role),
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
	// 空闲超时，卸载 Actor
	case IdleTimeout:
		ctx.Self().Tell(actor.PoisonPill{})
		return
	// 查询会话基本信息
	case GetConvInfoQuery:
		ctx.Reply(Result{Data: a.toDTO()})
	// 更新会话名称和头像
	case UpdateConvInfoCmd:
		a.handleUpdateInfo(ctx, msg)
	// 删除会话
	case DeleteConvCmd:
		a.handleDelete(ctx, msg)
	// 查询会话成员列表
	case ListMembersQuery:
		a.handleListMembers(ctx)
	// 添加成员到会话
	case AddMemberCmd:
		a.handleAddMember(ctx, msg)
	// 从会话中移除成员
	case RemoveMemberCmd:
		a.handleRemoveMember(ctx, msg)
	// 成员主动退出会话
	case LeaveConvCmd:
		a.handleLeave(ctx, msg)
	// 设置成员角色
	case SetRoleCmd:
		a.handleSetRole(ctx, msg)
	// 转让群主身份
	case TransferOwnerCmd:
		a.handleTransferOwner(ctx, msg)
	// 解散群聊
	case DissolveConvCmd:
		a.handleDissolve(ctx, msg)
	// 置顶/取消置顶会话
	case PinConvCmd:
		a.handlePin(ctx, msg)
	// 免打扰/取消免打扰会话
	case MuteConvCmd:
		a.handleMute(ctx, msg)
	// 标记已读到指定seq
	case ReadConvCmd:
		a.handleRead(ctx, msg)
	// 标记会话全部已读
	case ReadAllConvCmd:
		a.handleReadAll(ctx, msg)
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

func (a *ConversationActor) handleUpdateInfo(ctx actor.Context, msg UpdateConvInfoCmd) {
	now := time.Now().Unix()
	if err := a.store.UpdateConversation(a.convID, map[string]any{"name": msg.Name, "avatar": msg.Avatar, "updated_at": now}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	a.name = msg.Name
	a.avatar = msg.Avatar
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleDelete(ctx actor.Context, _ DeleteConvCmd) {
	if err := a.store.DeleteConversation(a.convID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
	ctx.Self().Tell(actor.PoisonPill{})
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
	if _, exists := a.members[msg.UID]; exists {
		ctx.Reply(Result{Err: fmt.Errorf("member already exists")})
		return
	}
	if len(a.members) >= a.memberLimit {
		ctx.Reply(Result{Err: fmt.Errorf("member limit reached")})
		return
	}

	now := time.Now().Unix()
	member := &dal.Member{
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
}

func (a *ConversationActor) handleRemoveMember(ctx actor.Context, msg RemoveMemberCmd) {
	if _, exists := a.members[msg.UID]; !exists {
		ctx.Reply(Result{Err: fmt.Errorf("member not found")})
		return
	}
	if err := a.store.DeleteMember(a.convID, msg.UID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	delete(a.members, msg.UID)
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleLeave(ctx actor.Context, msg LeaveConvCmd) {
	if _, exists := a.members[msg.UID]; !exists {
		ctx.Reply(Result{Err: fmt.Errorf("not a member")})
		return
	}
	if err := a.store.DeleteMember(a.convID, msg.UID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	delete(a.members, msg.UID)
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleSetRole(ctx actor.Context, msg SetRoleCmd) {
	m, exists := a.members[msg.UID]
	if !exists {
		ctx.Reply(Result{Err: fmt.Errorf("member not found")})
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
	newOwner, exists := a.members[msg.NewOwnerID]
	if !exists {
		ctx.Reply(Result{Err: fmt.Errorf("new owner is not a member")})
		return
	}
	oldOwner := a.members[a.ownerID]

	if err := a.store.TransferOwner(a.convID, a.ownerID, msg.NewOwnerID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	oldOwner.Role = MemberRoleRegular
	newOwner.Role = MemberRoleOwner
	a.ownerID = msg.NewOwnerID
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleDissolve(ctx actor.Context, msg DissolveConvCmd) {
	if msg.OwnerID != a.ownerID {
		ctx.Reply(Result{Err: fmt.Errorf("only owner can dissolve")})
		return
	}
	if err := a.store.DeleteAllMembers(a.convID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.store.DeleteConversation(a.convID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
	ctx.Self().Tell(actor.PoisonPill{})
}

func (a *ConversationActor) handlePin(ctx actor.Context, msg PinConvCmd) {
	if err := a.store.UpdateUserConversation(msg.UID, a.convID, map[string]any{"is_pinned": msg.Pinned}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleMute(ctx actor.Context, msg MuteConvCmd) {
	if err := a.store.UpdateUserConversation(msg.UID, a.convID, map[string]any{"is_muted": msg.Muted}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleRead(ctx actor.Context, msg ReadConvCmd) {
	m, exists := a.members[msg.UID]
	if !exists {
		ctx.Reply(Result{Err: fmt.Errorf("not a member")})
		return
	}
	if msg.Seq <= m.LastReadSeq {
		ctx.Reply(Result{Data: true})
		return
	}
	if err := a.store.UpdateMember(a.convID, msg.UID, map[string]any{"last_read_seq": msg.Seq}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	m.LastReadSeq = msg.Seq
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) handleReadAll(ctx actor.Context, msg ReadAllConvCmd) {
	m, exists := a.members[msg.UID]
	if !exists {
		ctx.Reply(Result{Err: fmt.Errorf("not a member")})
		return
	}
	if err := a.store.UpdateMember(a.convID, msg.UID, map[string]any{"last_read_seq": a.maxSeq}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	m.LastReadSeq = a.maxSeq
	ctx.Reply(Result{Data: true})
}
