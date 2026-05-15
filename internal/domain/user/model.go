package user

type User struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	Username  string `gorm:"uniqueIndex;size:64;not null"`
	Password  string `gorm:"size:128;not null"`
	Nickname  string `gorm:"size:64;not null"`
	Avatar    string `gorm:"size:256"`
	Sign      string `gorm:"size:256"`
	Status    int8   `gorm:"default:0"`
	CreatedAt int64  `gorm:"not null"`
	UpdatedAt int64  `gorm:"not null"`
}
