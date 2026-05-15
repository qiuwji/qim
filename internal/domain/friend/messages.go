package friend

type FriendRequestStatus int8

const (
	FriendRequestPending FriendRequestStatus = iota
	FriendRequestAccepted
	FriendRequestRejected
)

type Result struct {
	Data any
	Err  error
}

type FriendRequestDTO struct {
	ID        uint64              `json:"id"`
	FromUID   uint64              `json:"from_uid"`
	ToUID     uint64              `json:"to_uid"`
	Message   string              `json:"message"`
	Status    FriendRequestStatus `json:"status"`
	CreatedAt int64               `json:"created_at"`
}

type FriendDTO struct {
	ID        uint64 `json:"id"`
	FriendUID uint64 `json:"friend_uid"`
	Remark    string `json:"remark"`
	GroupID   uint64 `json:"group_id"`
	CreatedAt int64  `json:"created_at"`
}

type FriendGroupDTO struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type SendRequestCmd struct {
	FromUID uint64
	ToUID   uint64
	Message string
}

type ListIncomingCmd struct {
	UID uint64
}

type ListOutgoingCmd struct {
	UID uint64
}

type HandleRequestCmd struct {
	UID    uint64
	ReqID  uint64
	Accept bool
}

type DeleteFriendCmd struct {
	UID       uint64
	FriendUID uint64
}

type ListFriendsCmd struct {
	UID uint64
}

type UpdateRemarkCmd struct {
	UID       uint64
	FriendUID uint64
	Remark    string
}

type MoveGroupCmd struct {
	UID       uint64
	FriendUID uint64
	GroupID   uint64
}

type ListGroupsCmd struct {
	UID uint64
}

type CreateGroupCmd struct {
	UID  uint64
	Name string
}

type RenameGroupCmd struct {
	UID     uint64
	GroupID uint64
	Name    string
}

type DeleteGroupCmd struct {
	UID     uint64
	GroupID uint64
}

type SortGroupsCmd struct {
	UID    uint64
	Groups []struct {
		GroupID   uint64
		SortOrder int
	}
}
