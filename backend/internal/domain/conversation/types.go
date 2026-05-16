package conversation

import "qim/internal/domain/conversation/store"

type ConvType = store.ConvType
type MemberRole = store.MemberRole

const (
	ConvTypePrivate = store.ConvTypePrivate
	ConvTypeGroup   = store.ConvTypeGroup
)

const (
	MemberRoleRegular = store.MemberRoleRegular
	MemberRoleAdmin   = store.MemberRoleAdmin
	MemberRoleOwner   = store.MemberRoleOwner
)
