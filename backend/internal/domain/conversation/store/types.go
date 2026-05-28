package store

type ConvType int8

const (
	ConvTypePrivate    ConvType = iota + 1
	ConvTypeGroup
	ConvTypeBotSession // multi-session bot conversation (user ↔ bot, one per session)
)

type MemberRole int8

const (
	MemberRoleRegular MemberRole = iota
	MemberRoleAdmin
	MemberRoleOwner
)
