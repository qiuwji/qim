package actor

// DeadLetter 描述一条未能留在目标邮箱中的消息。
type DeadLetter struct {
	Target   string
	Envelope Envelope
	Reason   MailboxDropReason
}

// DeadLetterHandler 允许调用方接管死信处理，例如写日志、转储或进入专门队列。
type DeadLetterHandler func(DeadLetter)
