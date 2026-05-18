package conversation

import "qim/internal/domain/conversation/store"

const (
	MsgTypeText   int8 = 1
	MsgTypeSystem int8 = 5
)

type Result struct {
	Data any
	Err  error
}

type ConversationDTO struct {
	ID          uint64         `json:"id"`
	Type        store.ConvType `json:"type"`
	Name        string         `json:"name"`
	Avatar      string         `json:"avatar"`
	OwnerID     uint64         `json:"owner_id"`
	MemberCount int            `json:"member_count"`
	MemberLimit int            `json:"member_limit"`
	MaxSeq      int64          `json:"max_seq"`
	CreatedAt   int64          `json:"created_at"`
}

type MemberDTO struct {
	UID         uint64           `json:"uid"`
	Role        store.MemberRole `json:"role"`
	LastReadSeq int64            `json:"last_read_seq"`
	JoinTime    int64            `json:"join_time"`
}

type UserConvDTO struct {
	ConversationID uint64           `json:"conversation_id"`
	IsPinned       bool             `json:"is_pinned"`
	IsMuted        bool             `json:"is_muted"`
	UnreadCount    int              `json:"unread_count"`
	LastMsgAt      int64            `json:"last_msg_at"`
	Conv           *ConversationDTO `json:"conv,omitempty"`
}

type MessageDTO struct {
	ID             uint64   `json:"id"`
	ConversationID uint64   `json:"conversation_id"`
	Seq            int64    `json:"seq"`
	SenderID       uint64   `json:"sender_id"`
	MsgType        int8     `json:"msg_type"`
	Content        string   `json:"content"`
	ReplyTo        uint64   `json:"reply_to"`
	MentionUIDs    []uint64 `json:"mention_uids"`
	MentionAll     bool     `json:"mention_all"`
	Revoked        bool     `json:"revoked"`
	ClientID       string   `json:"client_id"`
	CreatedAt      int64    `json:"created_at"`
}

type ListUserConversationsCmd struct {
	UID uint64
}

type CreatePrivateConvCmd struct {
	UID1 uint64
	UID2 uint64
}

type CreateGroupConvCmd struct {
	OwnerID uint64
	Name    string
	Avatar  string
	Members []uint64
}

type SendMessageCmd struct {
	SenderID    uint64
	MsgType     int8
	Content     string
	ReplyTo     uint64
	ClientID    string
	MentionUIDs []uint64
	MentionAll  bool
}

type RevokeMessageCmd struct {
	OperatorID uint64
	MessageID  uint64
}

type TypingCmd struct {
	UID uint64
}

type TypingPushCmd struct {
	ConversationID uint64 `json:"conversation_id"`
	FromUID        uint64 `json:"user_id"`
	ToUID          uint64 `json:"-"`
}

type GetConvInfoQuery struct{}

type UpdateConvInfoCmd struct {
	OperatorID  uint64
	Name        *string
	Avatar      *string
	MemberLimit *int
}

type ListMembersQuery struct{}

type AddMemberCmd struct {
	OperatorID uint64
	UID        uint64
	Role       store.MemberRole
}

type RemoveMemberCmd struct {
	OperatorID uint64
	UID        uint64
}

type LeaveConvCmd struct {
	UID uint64
}

type SetRoleCmd struct {
	OperatorID uint64
	UID        uint64
	Role       store.MemberRole
}

type TransferOwnerCmd struct {
	OperatorID uint64
	NewOwnerID uint64
}

type DissolveConvCmd struct {
	OperatorID uint64
}

type PinConvCmd struct {
	UID            uint64
	ConversationID uint64
	Pinned         bool
}

type MuteConvCmd struct {
	UID            uint64
	ConversationID uint64
	Muted          bool
}

type ReadConvCmd struct {
	UID            uint64
	ConversationID uint64
	Seq            int64
}

type ReadAllConvCmd struct {
	UID uint64
}

type IdleTimeout struct{}
