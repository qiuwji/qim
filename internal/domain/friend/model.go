package friend

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
}

type Friend struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"uniqueIndex:idx_user_friend;not null"`
	FriendUID uint64 `gorm:"uniqueIndex:idx_user_friend;not null"`
	Remark    string `gorm:"size:64"`
	GroupID   uint64
	CreatedAt int64 `gorm:"not null"`
}
