package user

type UserStatus int8

const (
	UserStatusNormal UserStatus = iota
	UserStatusBanned
)

type Result struct {
	Data any
	Err  error
}

type UserDTO struct {
	ID        uint64     `json:"id"`
	Username  string     `json:"username"`
	Nickname  string     `json:"nickname"`
	Avatar    string     `json:"avatar"`
	Sign      string     `json:"sign"`
	Status    UserStatus `json:"status"`
	CreatedAt int64      `json:"created_at"`
}

type LoginResult struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

type RegisterCmd struct {
	Username string
	Password string
	Nickname string
}

type LoginCmd struct {
	Username string
	Password string
}

type SearchUsersCmd struct {
	Keyword string
}

type GetUserCmd struct {
	UID uint64
}

type GetProfileQuery struct{}

type UpdateProfileCmd struct {
	Nickname string
	Avatar   string
	Sign     string
}

type ChangePasswordCmd struct {
	OldPassword string
	NewPassword string
}
