package actor

import "sync"

// BackpressurePolicy 是旧版背压策略枚举，保留它以兼容简单配置入口。
type BackpressurePolicy int

const (
	// BackpressureBlock 在邮箱满载时阻塞发送方，直到有空间或 actor 停止。
	BackpressureBlock BackpressurePolicy = iota
	// BackpressureDropNewest 在邮箱满载时直接丢弃当前新消息。
	BackpressureDropNewest
	// BackpressureDropOldest 在邮箱满载时淘汰最旧消息，为新消息腾出空间。
	BackpressureDropOldest
)

// MailboxAction 描述邮箱饱和时的策略动作。
type MailboxAction int

const (
	// MailboxActionBlock 阻塞等待队列腾出空间。
	MailboxActionBlock MailboxAction = iota
	// MailboxActionDropNewest 丢弃当前要入队的新消息。
	MailboxActionDropNewest
	// MailboxActionDropOldest 丢弃队列中最旧的消息，再接收新消息。
	MailboxActionDropOldest
	// MailboxActionReject 拒绝新消息，但不淘汰现有队列内容。
	MailboxActionReject
)

// MailboxDropReason 描述消息未能留在邮箱中的原因。
type MailboxDropReason string

const (
	MailboxDropReasonNone    MailboxDropReason = ""
	MailboxDropReasonStopped MailboxDropReason = "stopped"
	MailboxDropReasonFull    MailboxDropReason = "full"
	MailboxDropReasonEvicted MailboxDropReason = "evicted"
	MailboxDropReasonPolicy  MailboxDropReason = "policy_reject"
)

// MailboxState 描述邮箱当前的容量状态，供策略决策使用。
type MailboxState struct {
	Length   int
	Capacity int
	Stopped  bool
}

// MailboxDecision 描述策略对“邮箱已满”场景的处理决定。
type MailboxDecision struct {
	Action MailboxAction
	Reason MailboxDropReason
}

// MailboxPolicy 允许调用方定制邮箱饱和时的处理行为。
type MailboxPolicy interface {
	OnFull(state MailboxState, incoming Envelope) MailboxDecision
}

// MailboxPolicyFunc 允许用函数快速定义策略。
type MailboxPolicyFunc func(state MailboxState, incoming Envelope) MailboxDecision

func (f MailboxPolicyFunc) OnFull(state MailboxState, incoming Envelope) MailboxDecision {
	return f(state, incoming)
}

// MailboxDrop 描述一次入队过程里被丢弃的消息。
type MailboxDrop struct {
	Envelope Envelope
	Reason   MailboxDropReason
}

// MailboxPushResult 描述一次入队的结果。
type MailboxPushResult struct {
	Accepted   bool
	QueueDepth int
	Reason     MailboxDropReason
	Dropped    *MailboxDrop
}

// Mailbox 是 actor 邮箱接口，负责消息的入队和出队。
type Mailbox interface {
	Push(e Envelope) MailboxPushResult
	Pull() (Envelope, bool)
	Stop()
	Len() int
}

// MailboxFactory 创建新的 Mailbox 实例。
type MailboxFactory func() Mailbox

// ChanMailbox 是基于内存队列和策略插件的默认邮箱实现。
type ChanMailbox struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	queue    []Envelope
	capacity int
	stopped  bool
	policy   MailboxPolicy
}

// NewChanMailbox 创建指定缓冲区大小的默认邮箱。
func NewChanMailbox(size int) *ChanMailbox {
	return NewChanMailboxWithPolicy(size, NewBlockingMailboxPolicy())
}

// NewChanMailboxWithPolicy 创建带策略插件的默认邮箱。
func NewChanMailboxWithPolicy(size int, policy MailboxPolicy) *ChanMailbox {
	if size <= 0 {
		size = 1
	}
	if policy == nil {
		policy = NewBlockingMailboxPolicy()
	}

	m := &ChanMailbox{
		queue:    make([]Envelope, 0, size),
		capacity: size,
		policy:   policy,
	}
	m.notEmpty = sync.NewCond(&m.mu)
	m.notFull = sync.NewCond(&m.mu)
	return m
}

// NewBlockingMailboxPolicy 返回阻塞等待型策略。
func NewBlockingMailboxPolicy() MailboxPolicy {
	return MailboxPolicyFunc(func(state MailboxState, incoming Envelope) MailboxDecision {
		return MailboxDecision{Action: MailboxActionBlock}
	})
}

// NewDropNewestMailboxPolicy 返回丢弃新消息型策略。
func NewDropNewestMailboxPolicy() MailboxPolicy {
	return MailboxPolicyFunc(func(state MailboxState, incoming Envelope) MailboxDecision {
		return MailboxDecision{
			Action: MailboxActionDropNewest,
			Reason: MailboxDropReasonFull,
		}
	})
}

// NewDropOldestMailboxPolicy 返回淘汰最旧消息型策略。
func NewDropOldestMailboxPolicy() MailboxPolicy {
	return MailboxPolicyFunc(func(state MailboxState, incoming Envelope) MailboxDecision {
		return MailboxDecision{
			Action: MailboxActionDropOldest,
			Reason: MailboxDropReasonEvicted,
		}
	})
}

func (m *ChanMailbox) Push(e Envelope) MailboxPushResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		if m.stopped {
			return MailboxPushResult{
				Reason:     MailboxDropReasonStopped,
				QueueDepth: len(m.queue),
				Dropped: &MailboxDrop{
					Envelope: e,
					Reason:   MailboxDropReasonStopped,
				},
			}
		}

		if len(m.queue) < m.capacity {
			m.queue = append(m.queue, e)
			m.notEmpty.Signal()
			return MailboxPushResult{
				Accepted:   true,
				QueueDepth: len(m.queue),
			}
		}

		decision := m.policy.OnFull(m.snapshotLocked(), e)
		decision = normalizeDecision(decision)

		switch decision.Action {
		case MailboxActionBlock:
			m.notFull.Wait()
		case MailboxActionDropNewest, MailboxActionReject:
			return MailboxPushResult{
				Reason:     decision.Reason,
				QueueDepth: len(m.queue),
				Dropped: &MailboxDrop{
					Envelope: e,
					Reason:   decision.Reason,
				},
			}
		case MailboxActionDropOldest:
			dropped := m.queue[0]
			m.queue = append(m.queue[1:], e)
			m.notEmpty.Signal()
			return MailboxPushResult{
				Accepted:   true,
				QueueDepth: len(m.queue),
				Dropped: &MailboxDrop{
					Envelope: dropped,
					Reason:   decision.Reason,
				},
			}
		default:
			m.notFull.Wait()
		}
	}
}

func (m *ChanMailbox) Pull() (Envelope, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for len(m.queue) == 0 && !m.stopped {
		m.notEmpty.Wait()
	}
	if len(m.queue) == 0 {
		return Envelope{}, false
	}

	env := m.queue[0]
	m.queue[0] = Envelope{}
	m.queue = m.queue[1:]
	m.notFull.Signal()
	return env, true
}

func (m *ChanMailbox) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return
	}
	m.stopped = true
	m.notEmpty.Broadcast()
	m.notFull.Broadcast()
}

func (m *ChanMailbox) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.queue)
}

func (m *ChanMailbox) snapshotLocked() MailboxState {
	return MailboxState{
		Length:   len(m.queue),
		Capacity: m.capacity,
		Stopped:  m.stopped,
	}
}

func normalizeDecision(decision MailboxDecision) MailboxDecision {
	if decision.Action == MailboxActionDropNewest && decision.Reason == MailboxDropReasonNone {
		decision.Reason = MailboxDropReasonFull
	}
	if decision.Action == MailboxActionReject && decision.Reason == MailboxDropReasonNone {
		decision.Reason = MailboxDropReasonPolicy
	}
	if decision.Action == MailboxActionDropOldest && decision.Reason == MailboxDropReasonNone {
		decision.Reason = MailboxDropReasonEvicted
	}
	return decision
}
