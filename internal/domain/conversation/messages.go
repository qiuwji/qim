package conversation

type Result struct {
	Data any
	Err  error
}

type ConversationDTO struct {
	ID          uint64   `json:"id"`
	Type        ConvType `json:"type"`
	Name        string   `json:"name"`
	Avatar      string   `json:"avatar"`
	OwnerID     uint64   `json:"owner_id"`
	MemberCount int      `json:"member_count"`
	MemberLimit int      `json:"member_limit"`
	MaxSeq      int64    `json:"max_seq"`
	CreatedAt   int64    `json:"created_at"`
}

type MemberDTO struct {
	UID         uint64     `json:"uid"`
	Role        MemberRole `json:"role"`
	LastReadSeq int64      `json:"last_read_seq"`
	JoinTime    int64      `json:"join_time"`
}

type UserConvDTO struct {
	ConversationID uint64           `json:"conversation_id"`
	IsPinned       bool             `json:"is_pinned"`
	IsMuted        bool             `json:"is_muted"`
	UnreadCount    int              `json:"unread_count"`
	LastMsgAt      int64            `json:"last_msg_at"`
	Conv           *ConversationDTO `json:"conv,omitempty"`
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

type GetConvInfoQuery struct{}

type UpdateConvInfoCmd struct {
	Name   string
	Avatar string
}

type DeleteConvCmd struct {
	UID uint64
}

type ListMembersQuery struct{}

type AddMemberCmd struct {
	UID  uint64
	Role MemberRole
}

type RemoveMemberCmd struct {
	UID uint64
}

type LeaveConvCmd struct {
	UID uint64
}

type SetRoleCmd struct {
	UID  uint64
	Role MemberRole
}

type TransferOwnerCmd struct {
	NewOwnerID uint64
}

type DissolveConvCmd struct {
	OwnerID uint64
}

type PinConvCmd struct {
	UID    uint64
	Pinned bool
}

type MuteConvCmd struct {
	UID   uint64
	Muted bool
}

type ReadConvCmd struct {
	UID uint64
	Seq int64
}

type ReadAllConvCmd struct {
	UID uint64
}

type IdleTimeout struct{}
