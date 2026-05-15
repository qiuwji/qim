package conversation

const EventMessageSent = "conversation.message_sent"

type MessageSentEvent struct {
	MessageID      uint64
	ConversationID uint64
	Seq            int64
	SenderID       uint64
	MemberUIDs     []uint64
	MsgType        int8
	Content        string
	ReplyTo        uint64
	ClientID       string
	CreatedAt      int64
}

func (MessageSentEvent) Name() string {
	return EventMessageSent
}
