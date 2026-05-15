package conversation

type ConvType int8

const (
	ConvTypePrivate ConvType = iota + 1
	ConvTypeGroup
)

type MemberRole int8

const (
	MemberRoleRegular MemberRole = iota
	MemberRoleAdmin
	MemberRoleOwner
)
