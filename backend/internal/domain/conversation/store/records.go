package store

type CreatePrivateConversationInput struct {
	UID1      uint64
	UID2      uint64
	CreatedAt int64
}

type CreateGroupConversationInput struct {
	OwnerID    uint64
	Name       string
	Avatar     string
	MemberUIDs []uint64
	CreatedAt  int64
}

type ConversationRecord struct {
	ID          uint64
	Type        int8
	Name        string
	Avatar      string
	OwnerID     uint64
	MaxSeq      int64
	MemberLimit int
	CreatedAt   int64
	UpdatedAt   int64
}

type MemberRecord struct {
	ConversationID uint64
	UserID         uint64
	Role           int8
	LastReadSeq    int64
	JoinTime       int64
}

type UserConversationRecord struct {
	ConversationID uint64
	IsPinned       bool
	IsMuted        bool
	UnreadCount    int
	LastMsgAt      int64
}

type MessageCommitInput struct {
	Message          MessageAppendInput
	UnreadProjection UnreadProjectionInput
}

type MessageAppendInput struct {
	ConversationID uint64
	Seq            int64
	SenderID       uint64
	MsgType        int8
	Content        string
	ReplyTo        uint64
	ClientID       string
	CreatedAt      int64
	MentionUIDs    []uint64
	MentionAll     bool
}

type UnreadProjectionInput struct {
	ConversationID uint64
	SenderID       uint64
	MemberUIDs     []uint64
	LastMsgAt      int64
}

type MessageCommitResult struct {
	MessageID  uint64
	Seq        int64
	SenderID   uint64
	MsgType    int8
	Content    string
	ReplyTo    uint64
	ClientID   string
	CreatedAt  int64
	Duplicated bool
	Revoked    bool
}

type MessageRecord struct {
	ID             uint64
	ConversationID uint64
	Seq            int64
	SenderID       uint64
	Revoked        bool
}
