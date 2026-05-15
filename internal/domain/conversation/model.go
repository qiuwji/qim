package conversation

type Conversation struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	Type        int8   `gorm:"not null"`
	Name        string `gorm:"size:64"`
	Avatar      string `gorm:"size:256"`
	OwnerID     uint64
	MaxSeq      int64 `gorm:"default:0"`
	MemberLimit int   `gorm:"default:500"`
	CreatedAt   int64 `gorm:"not null"`
	UpdatedAt   int64 `gorm:"not null"`
}

type Member struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64 `gorm:"uniqueIndex:idx_conv_user;not null"`
	UserID         uint64 `gorm:"uniqueIndex:idx_conv_user;not null"`
	Role           int8   `gorm:"default:0"`
	LastReadSeq    int64  `gorm:"default:0"`
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
