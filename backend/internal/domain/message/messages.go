package message

type MsgType int8

const (
	MsgTypeText MsgType = iota + 1
	MsgTypeImage
	MsgTypeFile
	MsgTypeVoice
	MsgTypeSystem
	MsgTypeCustom
)

type Result struct {
	Data any
	Err  error
}

type MessageDTO struct {
	ID             uint64   `json:"id"`
	ConversationID uint64   `json:"conversation_id"`
	Seq            int64    `json:"seq"`
	SenderID       uint64   `json:"sender_id"`
	MsgType        MsgType  `json:"msg_type"`
	Content        string   `json:"content"`
	ReplyTo        uint64   `json:"reply_to"`
	MentionUIDs    []uint64 `json:"mention_uids"`
	MentionAll     bool     `json:"mention_all"`
	Revoked        bool     `json:"revoked"`
	Edited         bool     `json:"edited"`
	ClientID       string   `json:"client_id"`
	CreatedAt      int64    `json:"created_at"`
}

type StoreMsgCmd struct {
	ConversationID uint64
	SenderID       uint64
	MsgType        MsgType
	Content        string
	ReplyTo        uint64
	ClientID       string
}

type ListMessagesCmd struct {
	ConversationID uint64
	BeforeSeq      int64
	Limit          int
}

type SearchMessagesCmd struct {
	ConversationID uint64
	Keyword        string
	Limit          int
}
