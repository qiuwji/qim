package dal

type Conversation struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	Type        int8   `gorm:"not null"`
	Name        string `gorm:"size:64"`
	Avatar      string `gorm:"size:256"`
	OwnerID     uint64
	MaxSeq      int64 `gorm:"default:0"`
	MemberLimit int   `gorm:"default:500"`
	Status      int8  `gorm:"default:0"`
	CreatedAt   int64 `gorm:"not null"`
	UpdatedAt   int64 `gorm:"not null"`
}

type Member struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64 `gorm:"uniqueIndex:idx_conv_user;not null"`
	UserID         uint64 `gorm:"uniqueIndex:idx_conv_user;not null"`
	Role           int8   `gorm:"default:0"`
	LastReadSeq    int64  `gorm:"default:0"`
	Status         int8   `gorm:"default:0"`
	JoinTime       int64  `gorm:"not null"`
}

type UserConversation struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	UserID         uint64 `gorm:"uniqueIndex:idx_user_conv;not null"`
	ConversationID uint64 `gorm:"uniqueIndex:idx_user_conv;not null"`
	IsPinned       bool   `gorm:"default:false"`
	IsMuted        bool   `gorm:"default:false"`
	IsDeleted      bool   `gorm:"default:false"`
	LastMsgAt      int64
	UnreadCount    int `gorm:"default:0"`
}

type User struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"uniqueIndex;size:64;not null"`
	Password     string `gorm:"size:128;not null"`
	Nickname     string `gorm:"size:64;not null"`
	Avatar       string `gorm:"size:256"`
	Sign         string `gorm:"size:256"`
	Status       int8   `gorm:"default:0"`
	UserType     int8   `gorm:"default:0"`
	CreatorUID   uint64 `gorm:"default:0;index"`
	CreatedAt    int64  `gorm:"not null"`
	UpdatedAt    int64  `gorm:"not null"`
	LastOnlineAt int64  `gorm:"not null"`
}

type Message struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64 `gorm:"uniqueIndex:idx_conv_seq;index:idx_msg_client_lookup;not null"`
	Seq            int64  `gorm:"uniqueIndex:idx_conv_seq;not null"`
	SenderID       uint64 `gorm:"index:idx_msg_client_lookup;not null"`
	MsgType        int8   `gorm:"not null"`
	Content        string `gorm:"type:text;not null"`
	ReplyTo        uint64 `gorm:"default:0"`
	MentionUIDs    string `gorm:"type:text"`
	MentionAll     bool   `gorm:"default:false"`
	Revoked        bool   `gorm:"default:false"`
	Edited         bool   `gorm:"default:false"`
	ClientID       string `gorm:"index:idx_msg_client_lookup;size:64"`
	CreatedAt      int64  `gorm:"not null"`
}

type FriendRequest struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	FromUID   uint64 `gorm:"index;not null"`
	ToUID     uint64 `gorm:"index;not null"`
	Message   string `gorm:"size:256"`
	Status    int8   `gorm:"default:0"`
	CreatedAt int64  `gorm:"not null"`
	UpdatedAt int64  `gorm:"not null"`
}

type FriendGroup struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"uniqueIndex:idx_user_name;not null"`
	Name      string `gorm:"uniqueIndex:idx_user_name;size:32;not null"`
	SortOrder int    `gorm:"default:0"`
	Status    int8   `gorm:"default:0"`
}

type Friend struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"uniqueIndex:idx_user_friend;not null"`
	FriendUID uint64 `gorm:"uniqueIndex:idx_user_friend;not null"`
	Remark    string `gorm:"size:64"`
	GroupID   uint64
	Status    int8  `gorm:"default:0"`
	CreatedAt int64 `gorm:"not null"`
}

type CallRecord struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	CallerUID uint64 `gorm:"index;not null"`
	CalleeUID uint64 `gorm:"index;not null"`
	CallType  int8   `gorm:"not null"`
	Status    int8   `gorm:"not null"`
	StartedAt int64
	EndedAt   int64 `gorm:"not null"`
	Duration  int64
	EndReason string `gorm:"size:32"`
	CreatedAt int64  `gorm:"not null"`
}

type BotConfig struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	UID         uint64 `gorm:"uniqueIndex;not null"`
	Permissions string `gorm:"type:text;default:'[]'"`
	CreatedAt   int64  `gorm:"not null"`
	UpdatedAt   int64  `gorm:"not null"`
}

type AgentSession struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	SessionID    string `gorm:"uniqueIndex;size:64;not null"`
	PlatformName string `gorm:"size:64"`
	Status       int8   `gorm:"default:0"`
	LastSeenAt   int64  `gorm:"index;not null"`
	ExpiresAt    int64  `gorm:"index;not null"`
	CreatedAt    int64  `gorm:"not null"`
	UpdatedAt    int64  `gorm:"not null"`
}

type AgentSubscription struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	SessionID  string `gorm:"index:idx_agent_sub_session_event;size:64;not null"`
	BotUID     uint64 `gorm:"index:idx_agent_sub_session_event;not null"`
	EventName  string `gorm:"index:idx_agent_sub_session_event;size:64;not null"`
	FilterJSON string `gorm:"type:text;default:''"`
	CreatedAt  int64  `gorm:"not null"`
	UpdatedAt  int64  `gorm:"not null"`
}

type AgentApproval struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ApprovalID     string `gorm:"uniqueIndex;size:64;not null"`
	SessionID      string `gorm:"index;size:64;not null"`
	BotUID         uint64 `gorm:"index;not null"`
	OwnerUID       uint64 `gorm:"index;not null"`
	ConversationID uint64
	Action         string `gorm:"size:128;not null"`
	Detail         string `gorm:"type:text;not null"`
	Status         int8   `gorm:"default:0"`
	ExpiresAt      int64  `gorm:"index;not null"`
	ResolvedAt     int64
	CreatedAt      int64 `gorm:"not null"`
	UpdatedAt      int64 `gorm:"not null"`
}
