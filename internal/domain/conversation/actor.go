package conversation

import (
	"context"
	"time"

	"qim/internal/actor"
	"qim/internal/eventbus"
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
	store       Store
	engine      *actor.Engine
	events      eventbus.Bus
	idleTimer   *actor.Timer
}

type MemberState struct {
	UID         uint64
	Role        MemberRole
	LastReadSeq int64
	JoinTime    int64
}

func NewConversationActor(convID uint64, store Store, engine *actor.Engine, events eventbus.Bus) *ConversationActor {
	return &ConversationActor{
		convID:  convID,
		members: make(map[uint64]*MemberState),
		store:   store,
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
	// 会话内发消息：由聚合根统一校验成员、分配 seq、提交事务
	case SendMessageCmd:
		a.handleSendMessage(ctx, msg)
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

func (a *ConversationActor) handleSendMessage(ctx actor.Context, msg SendMessageCmd) {
	if msg.Content == "" {
		ctx.Reply(Result{Err: ErrEmptyMessage})
		return
	}
	if _, exists := a.members[msg.SenderID]; !exists {
		ctx.Reply(Result{Err: ErrNotMember})
		return
	}

	now := time.Now().Unix()
	nextSeq := a.maxSeq + 1
	memberUIDs := a.memberUIDs()

	result, err := a.store.CommitMessage(MessageCommitInput{
		Message: MessageAppendInput{
			ConversationID: a.convID,
			Seq:            nextSeq,
			SenderID:       msg.SenderID,
			MsgType:        msg.MsgType,
			Content:        msg.Content,
			ReplyTo:        msg.ReplyTo,
			ClientID:       msg.ClientID,
			CreatedAt:      now,
		},
		UnreadProjection: UnreadProjectionInput{
			ConversationID: a.convID,
			SenderID:       msg.SenderID,
			MemberUIDs:     memberUIDs,
			LastMsgAt:      now,
		},
	})
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	a.maxSeq = nextSeq
	a.publishMessageSent(result.MessageID, nextSeq, msg, memberUIDs, now)
	ctx.Reply(Result{Data: MessageDTO{
		ID:             result.MessageID,
		ConversationID: a.convID,
		Seq:            nextSeq,
		SenderID:       msg.SenderID,
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		ReplyTo:        msg.ReplyTo,
		ClientID:       msg.ClientID,
		CreatedAt:      now,
	}})
}

func (a *ConversationActor) publishMessageSent(messageID uint64, seq int64, msg SendMessageCmd, memberUIDs []uint64, createdAt int64) {
	if a.events == nil {
		return
	}
	a.events.Publish(context.Background(), MessageSentEvent{
		MessageID:      messageID,
		ConversationID: a.convID,
		Seq:            seq,
		SenderID:       msg.SenderID,
		MemberUIDs:     append([]uint64(nil), memberUIDs...),
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		ReplyTo:        msg.ReplyTo,
		ClientID:       msg.ClientID,
		CreatedAt:      createdAt,
	})
}

func (a *ConversationActor) memberUIDs() []uint64 {
	uids := make([]uint64, 0, len(a.members))
	for uid := range a.members {
		uids = append(uids, uid)
	}
	return uids
}

func (a *ConversationActor) handleAddMember(ctx actor.Context, msg AddMemberCmd) {
	if _, exists := a.members[msg.UID]; exists {
		ctx.Reply(Result{Err: ErrMemberExists})
		return
	}
	if len(a.members) >= a.memberLimit {
		ctx.Reply(Result{Err: ErrMemberLimitReached})
		return
	}

	now := time.Now().Unix()
	member := &MemberRecord{
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
		ctx.Reply(Result{Err: ErrMemberNotFound})
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
		ctx.Reply(Result{Err: ErrNotMember})
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
		ctx.Reply(Result{Err: ErrMemberNotFound})
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
		ctx.Reply(Result{Err: ErrMemberNotFound})
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
		ctx.Reply(Result{Err: ErrOwnerRequired})
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
		ctx.Reply(Result{Err: ErrNotMember})
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
		ctx.Reply(Result{Err: ErrNotMember})
		return
	}
	if err := a.store.UpdateMember(a.convID, msg.UID, map[string]any{"last_read_seq": a.maxSeq}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	m.LastReadSeq = a.maxSeq
	ctx.Reply(Result{Data: true})
}
