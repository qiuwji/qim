package message

import (
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

type MessageStoreActor struct {
	store  dal.MsgStore
	engine *actor.Engine
}

func NewMessageStoreActor(store dal.MsgStore, engine *actor.Engine) *MessageStoreActor {
	return &MessageStoreActor{store: store, engine: engine}
}

func (a *MessageStoreActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case StoreMsgCmd:
		a.handleStore(ctx, msg)
	case ListMessagesCmd:
		a.handleList(ctx, msg)
	case SearchMessagesCmd:
		a.handleSearch(ctx, msg)
	}
}

func (a *MessageStoreActor) handleStore(ctx actor.Context, msg StoreMsgCmd) {
	now := time.Now().Unix()
	m := &dal.Message{
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		MsgType:        int8(msg.MsgType),
		Content:        msg.Content,
		ReplyTo:        msg.ReplyTo,
		ClientID:       msg.ClientID,
		CreatedAt:      now,
	}
	if err := a.store.CreateMessage(m); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		MsgType:        MsgType(m.MsgType),
		Content:        m.Content,
		ReplyTo:        m.ReplyTo,
		ClientID:       m.ClientID,
		CreatedAt:      m.CreatedAt,
	}})
}

func (a *MessageStoreActor) handleList(ctx actor.Context, msg ListMessagesCmd) {
	messages, err := a.store.ListMessages(msg.ConversationID, msg.BeforeSeq, msg.Limit)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]MessageDTO, 0, len(messages))
	for _, m := range messages {
		dtos = append(dtos, toDTO(m))
	}
	ctx.Reply(Result{Data: dtos})
}

func (a *MessageStoreActor) handleSearch(ctx actor.Context, msg SearchMessagesCmd) {
	messages, err := a.store.SearchMessages(msg.ConversationID, msg.Keyword, msg.Limit)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	dtos := make([]MessageDTO, 0, len(messages))
	for _, m := range messages {
		dtos = append(dtos, toDTO(m))
	}
	ctx.Reply(Result{Data: dtos})
}

func toDTO(m dal.Message) MessageDTO {
	dto := MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Seq:            m.Seq,
		SenderID:       m.SenderID,
		MsgType:        MsgType(m.MsgType),
		ReplyTo:        m.ReplyTo,
		Revoked:        m.Revoked,
		Edited:         m.Edited,
		ClientID:       m.ClientID,
		CreatedAt:      m.CreatedAt,
	}
	if !m.Revoked {
		dto.Content = m.Content
	}
	return dto
}
