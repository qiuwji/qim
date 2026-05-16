package actor

// Envelope 是消息信封，包装消息本身及其元数据
type Envelope struct {
	Sender  *ActorRef
	Message any
	replyTo *future
}
