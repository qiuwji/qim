package conversation

type Store interface {
	GetConversation(id uint64) (*ConversationRecord, error)
	CreateConversation(conv *ConversationRecord) error
	CreatePrivateConversation(input CreatePrivateConversationInput) (*ConversationRecord, error)
	CreateGroupConversation(input CreateGroupConversationInput) (*ConversationRecord, error)
	UpdateConversation(id uint64, updates map[string]any) error
	DissolveConversation(id uint64) error
	FindPrivateConversation(uid1, uid2 uint64) (*ConversationRecord, error)

	GetMembers(convID uint64) ([]MemberRecord, error)
	CreateMember(member *MemberRecord) error
	CreateMembers(members []MemberRecord) error
	DeleteMember(convID, uid uint64) error
	UpdateMember(convID, uid uint64, updates map[string]any) error
	TransferOwner(convID, oldOwnerUID, newOwnerUID uint64) error

	GetUserConversations(uid uint64) ([]UserConversationRecord, error)
	UpdateUserConversation(uid, convID uint64, updates map[string]any) error
	CommitMessage(input MessageCommitInput) (*MessageCommitResult, error)
	GetMessage(convID, messageID uint64) (*MessageRecord, error)
	RevokeMessage(convID, messageID uint64) error
}

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
}

type MessageRecord struct {
	ID             uint64
	ConversationID uint64
	Seq            int64
	SenderID       uint64
	Revoked        bool
}
