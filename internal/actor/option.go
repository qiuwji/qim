package actor

import "time"

// SupervisionStrategy 定义 actor 发生 panic 时的处理策略
type SupervisionStrategy int

const (
	// StrategyResume 恢复 panic 并继续处理下一条消息
	StrategyResume SupervisionStrategy = iota
	// StrategyStop 恢复 panic 并停止 actor
	StrategyStop
	// StrategyRestart 恢复 panic 并尝试用新实例重启 actor
	StrategyRestart
)

// SupervisionConfig 定义 actor 的 panic 处理与重启节流策略
type SupervisionConfig struct {
	Strategy       SupervisionStrategy
	MaxRestarts    int
	RestartBackoff time.Duration
	RestartWindow  time.Duration
}

// EngineOption 是 Engine 的配置选项函数
type EngineOption func(*Engine)

// WithLogger 设置自定义日志器
func WithLogger(l Logger) EngineOption {
	return func(e *Engine) {
		e.logger = l
	}
}

// WithMiddleware 添加消息处理中间件，按传入顺序依次包裹
func WithMiddleware(m ...Middleware) EngineOption {
	return func(e *Engine) {
		e.middleware = append(e.middleware, m...)
	}
}

// WithMailboxFactory 设置自定义邮箱工厂
func WithMailboxFactory(f MailboxFactory) EngineOption {
	return func(e *Engine) {
		e.mailboxFactory = f
	}
}

// WithMailboxSize 设置基于通道的邮箱缓冲区大小
func WithMailboxSize(size int) EngineOption {
	return func(e *Engine) {
		e.mailboxSize = size
	}
}

// WithMailboxPolicy 设置默认邮箱的饱和策略。
func WithMailboxPolicy(policy MailboxPolicy) EngineOption {
	return func(e *Engine) {
		e.mailboxPolicy = policy
	}
}

// WithBackpressurePolicy 设置默认通道邮箱的兼容背压策略。
func WithBackpressurePolicy(policy BackpressurePolicy) EngineOption {
	return func(e *Engine) {
		e.mailboxPolicy = mailboxPolicyFromBackpressure(policy)
	}
}

// WithDeadLetterHandler 设置死信处理器。
func WithDeadLetterHandler(handler DeadLetterHandler) EngineOption {
	return func(e *Engine) {
		e.deadLetterHandler = handler
	}
}

// WithMetrics 设置自定义指标收集器
func WithMetrics(m Metrics) EngineOption {
	return func(e *Engine) {
		e.metrics = m
	}
}

// WithSupervisionStrategy 设置 actor panic 时的处理策略，默认为 StrategyResume
func WithSupervisionStrategy(s SupervisionStrategy) EngineOption {
	return func(e *Engine) {
		e.supervision.Strategy = s
	}
}

// WithRestartPolicy 设置 restart 策略的节流配置
func WithRestartPolicy(maxRestarts int, backoff, window time.Duration) EngineOption {
	return func(e *Engine) {
		e.supervision.MaxRestarts = maxRestarts
		e.supervision.RestartBackoff = backoff
		e.supervision.RestartWindow = window
	}
}
