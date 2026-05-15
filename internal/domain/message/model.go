package message

type Message struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64 `gorm:"index;not null"`
	Seq            int64  `gorm:"not null"`
	SenderID       uint64 `gorm:"not null"`
	MsgType        int8   `gorm:"not null"`
	Content        string `gorm:"type:text;not null"`
	ReplyTo        uint64 `gorm:"default:0"`
	MentionUIDs    string `gorm:"size:512"`
	Revoked        bool   `gorm:"default:false"`
	Edited         bool   `gorm:"default:false"`
	ClientID       string `gorm:"index;size:64"`
	CreatedAt      int64  `gorm:"not null"`
}
