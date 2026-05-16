package conversation

import "qim/internal/domain/conversation/store"

type Store = store.Store

type CreatePrivateConversationInput = store.CreatePrivateConversationInput
type CreateGroupConversationInput = store.CreateGroupConversationInput
type ConversationRecord = store.ConversationRecord
type MemberRecord = store.MemberRecord
type UserConversationRecord = store.UserConversationRecord
type MessageCommitInput = store.MessageCommitInput
type MessageAppendInput = store.MessageAppendInput
type UnreadProjectionInput = store.UnreadProjectionInput
type MessageCommitResult = store.MessageCommitResult
type MessageRecord = store.MessageRecord
