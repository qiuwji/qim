package conversation

const (
	EventMessageSent         = "conversation.message_sent"
	EventMessageRevoked      = "conversation.message_revoked"
	EventMemberJoined        = "conversation.member_joined"
	EventMemberLeft          = "conversation.member_left"
	EventMemberKicked        = "conversation.member_kicked"
	EventOwnerTransferred    = "conversation.owner_transferred"
	EventGroupDissolved      = "conversation.group_dissolved"
	EventConversationUpdated = "conversation.updated"
)

type MessageSentEvent struct {
	MessageID      uint64
	ConversationID uint64
	ConvType       int8
	Seq            int64
	SenderID       uint64
	MemberUIDs     []uint64
	MsgType        int8
	Content        string
	ReplyTo        uint64
	ClientID       string
	CreatedAt      int64
	MentionUIDs    []uint64
	MentionAll     bool
}

func (MessageSentEvent) Name() string {
	return EventMessageSent
}

type MessageRevokedEvent struct {
	ConversationID uint64   `json:"conversation_id"`
	MessageID      uint64   `json:"message_id"`
	Seq            int64    `json:"seq"`
	SenderID       uint64   `json:"sender_id"`
	OperatorID     uint64   `json:"operator_id"`
	IsLatest       bool     `json:"is_latest"`
	MemberUIDs     []uint64 `json:"member_uids"`
}

func (MessageRevokedEvent) Name() string {
	return EventMessageRevoked
}

type ConversationUpdatedEvent struct {
	ConversationID uint64   `json:"conversation_id"`
	DisplayName    string   `json:"name,omitempty"`
	Avatar         string   `json:"avatar,omitempty"`
	MemberUIDs     []uint64 `json:"member_uids"`
}

func (ConversationUpdatedEvent) Name() string {
	return EventConversationUpdated
}

type MemberJoinedEvent struct {
	ConversationID uint64     `json:"conversation_id"`
	UID            uint64     `json:"uid"`
	Role           MemberRole `json:"role"`
	OperatorID     uint64     `json:"operator_id"`
	MemberUIDs     []uint64   `json:"member_uids"`
}

func (MemberJoinedEvent) Name() string {
	return EventMemberJoined
}

type MemberLeftEvent struct {
	ConversationID uint64   `json:"conversation_id"`
	UID            uint64   `json:"uid"`
	MemberUIDs     []uint64 `json:"member_uids"`
}

func (MemberLeftEvent) Name() string {
	return EventMemberLeft
}

type MemberKickedEvent struct {
	ConversationID uint64   `json:"conversation_id"`
	UID            uint64   `json:"uid"`
	OperatorID     uint64   `json:"operator_id"`
	MemberUIDs     []uint64 `json:"member_uids"`
}

func (MemberKickedEvent) Name() string {
	return EventMemberKicked
}

type OwnerTransferredEvent struct {
	ConversationID uint64   `json:"conversation_id"`
	OldOwnerID     uint64   `json:"old_owner_id"`
	NewOwnerID     uint64   `json:"new_owner_id"`
	MemberUIDs     []uint64 `json:"member_uids"`
}

func (OwnerTransferredEvent) Name() string {
	return EventOwnerTransferred
}

type GroupDissolvedEvent struct {
	ConversationID uint64   `json:"conversation_id"`
	OperatorID     uint64   `json:"operator_id"`
	MemberUIDs     []uint64 `json:"member_uids"`
}

func (GroupDissolvedEvent) Name() string {
	return EventGroupDissolved
}
