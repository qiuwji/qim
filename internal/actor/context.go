package actor

import (
	"time"
)

// Context 提供 actor 处理消息时的上下文信息和操作能力
type Context interface {
	// Self 返回当前 actor 的引用
	Self() *ActorRef
	// Sender 返回发送当前消息的 actor 引用，如果是外部发送则返回 nil
	Sender() *ActorRef
	// Message 返回当前正在处理的消息
	Message() any
	// Reply 向消息发送者回复消息，仅在 Ask 模式下有效
	Reply(msg any) error
	// Spawn 创建并启动一个新的子 actor
	Spawn(name string, a Actor) (*ActorRef, error)
	// Stop 停止指定的 actor
	Stop(ref *ActorRef)
	// Watch 监听指定 actor 的终止事件，当该 actor 停止时会收到 Terminated 消息
	Watch(ref *ActorRef)
	// Unwatch 取消对指定 actor 的监听
	Unwatch(ref *ActorRef)
	// ScheduleAfter 在指定时间后向当前 actor 发送一条消息，返回可取消的定时器
	ScheduleAfter(d time.Duration, msg any) *Timer
}

type actorContext struct {
	self    *ActorRef
	engine  *Engine
	mailbox Mailbox
	current *Envelope
	replied bool
}

func (c *actorContext) Self() *ActorRef {
	return c.self
}

func (c *actorContext) Sender() *ActorRef {
	if c.current == nil {
		return nil
	}
	return c.current.Sender
}

func (c *actorContext) Message() any {
	if c.current == nil {
		return nil
	}
	return c.current.Message
}

func (c *actorContext) Spawn(name string, a Actor) (*ActorRef, error) {
	return c.engine.Spawn(name, a)
}

func (c *actorContext) Stop(ref *ActorRef) {
	c.engine.Stop(ref.Name())
}

func (c *actorContext) Watch(ref *ActorRef) {
	if c.engine.watchManager.add(c.self, ref) {
		c.engine.metrics.WatchAdded(c.self.name, ref.name)
	}
}

func (c *actorContext) Unwatch(ref *ActorRef) {
	if c.engine.watchManager.remove(c.self, ref) {
		c.engine.metrics.WatchRemoved(c.self.name, ref.name)
	}
}

func (c *actorContext) ScheduleAfter(d time.Duration, msg any) *Timer {
	return c.engine.timerManager.Schedule(c.self, d, msg)
}

func (c *actorContext) Reply(msg any) error {
	if c.current == nil || c.current.replyTo == nil {
		return ErrReplyNoAsk
	}
	if c.replied {
		return ErrReplyDuplicate
	}
	c.replied = true
	c.current.replyTo.reply(msg)
	return nil
}
