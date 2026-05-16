package actor

// Terminated 是 actor 终止时发送给其监听者的信号
type Terminated struct {
	Who *ActorRef
}

// PoisonPill 是一种毒丸消息，actor 收到后会自动停止
type PoisonPill struct{}
