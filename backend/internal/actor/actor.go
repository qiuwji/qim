package actor

import (
	"time"
)

// Actor 是消息处理的核心接口，所有 actor 必须实现 Receive 方法
type Actor interface {
	Receive(ctx Context)
}

// ActorFactory 用于为 supervision restart 提供新的 actor 实例
type ActorFactory interface {
	NewActor() Actor
}

// Lifecycle 提供 actor 生命周期的钩子方法
type Lifecycle interface {
	OnStart(ctx Context)
	OnStop(ctx Context)
}

type deliverFunc func(Envelope) error

// ActorRef 是 actor 的引用，用于向 actor 发送消息
type ActorRef struct {
	name    string
	deliver deliverFunc
	done    chan struct{}
}

// Name 返回 actor 的名称
func (r *ActorRef) Name() string {
	return r.name
}

// Done 返回一个 channel，当 actor 停止时该 channel 会被关闭
func (r *ActorRef) Done() <-chan struct{} {
	return r.done
}

// Tell 向 actor 发送一条消息，如果 actor 已停止则返回错误
func (r *ActorRef) Tell(msg any) error {
	return r.deliver(Envelope{
		Sender:  nil,
		Message: msg,
	})
}

// TellFrom 以指定发送者的身份向 actor 发送一条消息
func (r *ActorRef) TellFrom(sender *ActorRef, msg any) error {
	return r.deliver(Envelope{
		Sender:  sender,
		Message: msg,
	})
}

// 约定消息类型 如果是Ask的消息就必须要返回响应
// Ask 向 actor 发送一条消息并等待回复，支持超时控制
func (r *ActorRef) Ask(msg any, timeout time.Duration) (any, error) {
	f := newFuture()
	if err := r.deliver(Envelope{
		Message: msg,
		replyTo: f,
	}); err != nil {
		return nil, err
	}
	return f.wait(timeout)
}
