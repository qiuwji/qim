package store

type Store interface {
	GetConversation(id uint64) (*ConversationRecord, error)
	CreateConversation(conv *ConversationRecord) error
	CreatePrivateConversation(input CreatePrivateConversationInput) (*ConversationRecord, error)
	CreateGroupConversation(input CreateGroupConversationInput) (*ConversationRecord, error)
	UpdateConversation(id uint64, updates map[string]any) error
	DissolveConversation(id uint64) error
	FindPrivateConversation(uid1, uid2 uint64) (*ConversationRecord, error)

	GetMembers(convID uint64) ([]MemberRecord, error)
	CreateMember(member *MemberRecord) error
	CreateMembers(members []MemberRecord) error
	DeleteMember(convID, uid uint64) error
	UpdateMember(convID, uid uint64, updates map[string]any) error
	TransferOwner(convID, oldOwnerUID, newOwnerUID uint64) error

	GetUserConversations(uid uint64) ([]UserConversationRecord, error)
	UpdateUserConversation(uid, convID uint64, updates map[string]any) error
	MarkConversationRead(uid, convID uint64, seq int64) error
	CommitMessage(input MessageCommitInput) (*MessageCommitResult, error)
	GetMessage(convID, messageID uint64) (*MessageRecord, error)
	RevokeMessage(convID, messageID uint64) error
}
