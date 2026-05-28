package conversation

import (
	"encoding/json"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation/store"
)

type callRecordContent struct {
	CallID    string `json:"call_id"`
	CallType  int8   `json:"call_type"`
	Duration  int64  `json:"duration"`
	EndReason string `json:"end_reason"`
	Status    int8   `json:"status"`
}

func (a *ConversationActor) handleAppendCallRecord(ctx actor.Context, msg AppendCallRecordCmd) {
	if len(a.members) == 0 {
		return
	}

	now := time.Now().Unix()
	nextSeq := a.maxSeq + 1
	memberUIDs := a.memberUIDs()

	content, _ := json.Marshal(callRecordContent{
		CallType:  msg.CallType,
		Duration:  msg.Duration,
		EndReason: msg.EndReason,
		Status:    msg.Status,
	})

	result, err := a.store.CommitMessage(store.MessageCommitInput{
		Message: store.MessageAppendInput{
			ConversationID: a.convID,
			Seq:            nextSeq,
			SenderID:       msg.CallerUID,
			MsgType:        MsgTypeCallRecord,
			Content:        string(content),
			CreatedAt:      now,
		},
		UnreadProjection: store.UnreadProjectionInput{
			ConversationID: a.convID,
			SenderID:       msg.CallerUID,
			MemberUIDs:     memberUIDs,
			LastMsgAt:      now,
		},
	})
	if err != nil {
		return
	}

	actualSeq := result.Seq
	if actualSeq == 0 {
		actualSeq = nextSeq
	}
	if !result.Duplicated {
		a.maxSeq = actualSeq
		a.publishMessageSent(result.MessageID, actualSeq, SendMessageCmd{
			SenderID: msg.CallerUID,
			MsgType:  MsgTypeCallRecord,
			Content:  string(content),
		}, nil, false, memberUIDs, now)
	}
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

	mentionUIDs, mentionAll := a.validateMention(msg)

	now := time.Now().Unix()
	nextSeq := a.maxSeq + 1
	memberUIDs := a.memberUIDs()

	result, err := a.store.CommitMessage(store.MessageCommitInput{
		Message: store.MessageAppendInput{
			ConversationID: a.convID,
			Seq:            nextSeq,
			SenderID:       msg.SenderID,
			MsgType:        msg.MsgType,
			Content:        msg.Content,
			ReplyTo:        msg.ReplyTo,
			ClientID:       msg.ClientID,
			CreatedAt:      now,
			MentionUIDs:    mentionUIDs,
			MentionAll:     mentionAll,
		},
		UnreadProjection: store.UnreadProjectionInput{
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

	actualSeq := result.Seq
	if actualSeq == 0 {
		actualSeq = nextSeq
	}
	createdAt := result.CreatedAt
	if createdAt == 0 {
		createdAt = now
	}
	senderID := result.SenderID
	if senderID == 0 {
		senderID = msg.SenderID
	}
	msgType := result.MsgType
	if msgType == 0 {
		msgType = msg.MsgType
	}
	content := result.Content
	if content == "" && !result.Revoked {
		content = msg.Content
	}
	replyTo := result.ReplyTo
	if replyTo == 0 {
		replyTo = msg.ReplyTo
	}
	clientID := result.ClientID
	if clientID == "" {
		clientID = msg.ClientID
	}
	if !result.Duplicated {
		a.maxSeq = actualSeq
		a.publishMessageSent(result.MessageID, actualSeq, msg, mentionUIDs, mentionAll, memberUIDs, createdAt)
	}
	ctx.Reply(Result{Data: MessageDTO{
		ID:             result.MessageID,
		ConversationID: a.convID,
		Seq:            actualSeq,
		SenderID:       senderID,
		MsgType:        msgType,
		Content:        content,
		ReplyTo:        replyTo,
		MentionUIDs:    mentionUIDs,
		MentionAll:     mentionAll,
		Revoked:        result.Revoked,
		ClientID:       clientID,
		CreatedAt:      createdAt,
	}})
}

func (a *ConversationActor) validateMention(msg SendMessageCmd) ([]uint64, bool) {
	if a.convType != store.ConvTypeGroup {
		return nil, false
	}
	if msg.MentionAll {
		if err := a.requireAdmin(msg.SenderID); err != nil {
			return a.filterMentionMembers(msg.MentionUIDs), false
		}
	}
	if len(msg.MentionUIDs) > 50 {
		return a.filterMentionMembers(msg.MentionUIDs[:50]), msg.MentionAll
	}
	return a.filterMentionMembers(msg.MentionUIDs), msg.MentionAll
}

func (a *ConversationActor) filterMentionMembers(uids []uint64) []uint64 {
	if len(uids) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(uids))
	result := make([]uint64, 0, len(uids))
	for _, uid := range uids {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		if _, isMember := a.members[uid]; !isMember {
			continue
		}
		seen[uid] = struct{}{}
		result = append(result, uid)
	}
	return result
}

func (a *ConversationActor) handleRevokeMessage(ctx actor.Context, msg RevokeMessageCmd) {
	if _, exists := a.members[msg.OperatorID]; !exists {
		ctx.Reply(Result{Err: ErrNotMember})
		return
	}
	record, err := a.store.GetMessage(a.convID, msg.MessageID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if record.Revoked {
		ctx.Reply(Result{Data: true})
		return
	}
	if record.SenderID != msg.OperatorID {
		if err := a.requireAdmin(msg.OperatorID); err != nil {
			ctx.Reply(Result{Err: err})
			return
		}
	}
	if err := a.store.RevokeMessage(a.convID, msg.MessageID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	a.publishMessageRevoked(record, msg.OperatorID)
	ctx.Reply(Result{Data: true})
}

func (a *ConversationActor) publishMessageSent(messageID uint64, seq int64, msg SendMessageCmd, mentionUIDs []uint64, mentionAll bool, memberUIDs []uint64, createdAt int64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(MessageSentEvent{
		MessageID:      messageID,
		ConversationID: a.convID,
		ConvType:       int8(a.convType),
		Seq:            seq,
		SenderID:       msg.SenderID,
		MemberUIDs:     append([]uint64(nil), memberUIDs...),
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		ReplyTo:        msg.ReplyTo,
		ClientID:       msg.ClientID,
		CreatedAt:      createdAt,
		MentionUIDs:    mentionUIDs,
		MentionAll:     mentionAll,
	})
}

func (a *ConversationActor) publishMessageRevoked(record *store.MessageRecord, operatorID uint64) {
	if a.events == nil {
		return
	}
	_ = a.events.Publish(MessageRevokedEvent{
		ConversationID: a.convID,
		MessageID:      record.ID,
		Seq:            record.Seq,
		SenderID:       record.SenderID,
		OperatorID:     operatorID,
		IsLatest:       record.Seq == a.maxSeq,
		MemberUIDs:     a.memberUIDs(),
	})
}
