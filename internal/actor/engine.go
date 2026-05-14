package actor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// actorInstance 是 actor 的运行时实例，持有 actor 对象、引用、上下文和处理函数
type actorInstance struct {
	actor          Actor
	ref            *ActorRef
	ctx            *actorContext
	receive        ReceiveFunc
	actorFactory   ActorFactory
	restartHistory []time.Time
}

func (inst *actorInstance) triggerOnStart() {
	if lifecycle, ok := inst.actor.(Lifecycle); ok {
		lifecycle.OnStart(inst.ctx)
	}
}

func (inst *actorInstance) triggerOnStop() {
	if lifecycle, ok := inst.actor.(Lifecycle); ok {
		lifecycle.OnStop(inst.ctx)
	}
}

func (inst *actorInstance) dispatch(env Envelope, onPanic func(any)) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			onPanic(r)
		}
	}()

	inst.ctx.current = &env
	inst.ctx.replied = false
	inst.receive(inst.ctx)
	inst.ctx.current = nil
	return false
}

// Engine 是 actor 系统的核心引擎，管理所有 actor 的生命周期
type Engine struct {
	actors            map[string]*actorInstance
	mu                sync.RWMutex
	logger            Logger
	metrics           Metrics
	middleware        []Middleware
	mailboxFactory    MailboxFactory
	mailboxSize       int
	mailboxPolicy     MailboxPolicy
	deadLetterHandler DeadLetterHandler
	watchManager      *watchManager
	timerManager      *timerManager
	supervision       SupervisionConfig
	shutdown          atomic.Bool
}

// NewEngine 创建一个新的 actor 引擎，支持通过选项函数进行配置
func NewEngine(opts ...EngineOption) *Engine {
	e := &Engine{
		actors:        make(map[string]*actorInstance),
		logger:        &defaultLogger{},
		metrics:       noopMetrics{},
		mailboxSize:   256,
		mailboxPolicy: NewBlockingMailboxPolicy(),
		watchManager:  newWatchManager(),
		supervision: SupervisionConfig{
			Strategy:      StrategyResume,
			MaxRestarts:   3,
			RestartWindow: 30 * time.Second,
		},
	}
	e.timerManager = newTimerManager(e.metrics)
	for _, opt := range opts {
		opt(e)
	}
	e.watchManager.setMetrics(e.metrics)
	e.timerManager.setMetrics(e.metrics)
	return e
}

// Metrics 返回引擎的指标收集器
func (e *Engine) Metrics() Metrics {
	return e.metrics
}

// Spawn 创建并启动一个新的 actor
func (e *Engine) Spawn(name string, a Actor) (*ActorRef, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.shutdown.Load() {
		return nil, ErrEngineShutdown
	}

	if _, exists := e.actors[name]; exists {
		return nil, ErrActorExists
	}

	inst := e.newActorInstance(name, a)
	e.actors[name] = inst
	e.metrics.ActorSpawned(name)
	e.metrics.ActorStateChanged(name, ActorStateStarting)
	go e.processLoop(inst)

	return inst.ref, nil
}

func (e *Engine) newActorInstance(name string, a Actor) *actorInstance {
	mailbox := e.newMailbox()
	ref := &ActorRef{
		name: name,
		done: make(chan struct{}),
	}
	ref.deliver = e.newDeliver(ref.name, mailbox)

	return &actorInstance{
		actor:        a,
		ref:          ref,
		ctx:          &actorContext{self: ref, engine: e, mailbox: mailbox},
		receive:      e.wrapReceive(a),
		actorFactory: resolveActorFactory(a),
	}
}

func (e *Engine) processLoop(inst *actorInstance) {
	defer func() {
		e.timerManager.cancelOwner(inst.ref)
		e.unregisterIfCurrent(inst)
		e.watchManager.cleanupForWatcher(inst.ref)
		inst.ctx.mailbox.Stop()
		e.metrics.ActorStateChanged(inst.ref.name, ActorStateStopped)
		e.metrics.ActorStopped(inst.ref.name)
		close(inst.ref.done)
	}()

	inst.triggerOnStart()
	e.metrics.ActorStateChanged(inst.ref.name, ActorStateRunning)

	for {
		env, ok := inst.ctx.mailbox.Pull()
		if !ok {
			e.beginStop(inst)
			return
		}
		e.metrics.MessageDequeued(inst.ref.name, inst.ctx.mailbox.Len())

		if _, ok := env.Message.(PoisonPill); ok {
			e.beginStop(inst)
			return
		}

		if !e.dispatchAndSupervise(inst, env) {
			e.beginStop(inst)
			return
		}
	}
}

func (e *Engine) dispatchAndSupervise(inst *actorInstance, env Envelope) bool {
	start := time.Now()
	panicked := inst.dispatch(env, func(r any) {
		e.logger.Printf("[actor] %s panic: %v", inst.ref.name, r)
		e.metrics.PanicRecovered(inst.ref.name)
	})
	e.metrics.MessageProcessed(inst.ref.name, time.Since(start))

	if env.replyTo != nil && !inst.ctx.replied {
		env.replyTo.replyError(ErrNoReply)
	}

	if !panicked {
		return true
	}

	switch e.supervision.Strategy {
	case StrategyResume:
		return true
	case StrategyRestart:
		if inst.restart(e) {
			return true
		}
	default:
	}
	return false
}

func (e *Engine) beginStop(inst *actorInstance) {
	e.metrics.ActorStateChanged(inst.ref.name, ActorStateStopping)
	inst.triggerOnStop()
}

func (e *Engine) unregisterIfCurrent(inst *actorInstance) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.actors[inst.ref.name] == inst {
		delete(e.actors, inst.ref.name)
	}
}

// Lookup 按名称查找 actor 引用
func (e *Engine) Lookup(name string) (*ActorRef, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	inst, ok := e.actors[name]
	if !ok {
		return nil, false
	}
	return inst.ref, true
}

// Stop 停止指定名称的 actor
func (e *Engine) Stop(name string) {
	inst := e.removeActor(name)
	if inst != nil {
		inst.ctx.mailbox.Stop()
	}
}

func (e *Engine) removeActor(name string) *actorInstance {
	e.mu.Lock()
	defer e.mu.Unlock()
	inst, ok := e.actors[name]
	if !ok {
		return nil
	}
	delete(e.actors, name)
	return inst
}

// Shutdown 优雅关闭引擎，等待所有 actor 停止
func (e *Engine) Shutdown() error {
	return e.ShutdownWithContext(context.Background())
}

// ShutdownWithContext 在指定上下文的截止时间内优雅关闭引擎
func (e *Engine) ShutdownWithContext(ctx context.Context) error {
	e.mu.Lock()
	e.shutdown.Store(true)
	instances := make([]*actorInstance, 0, len(e.actors))
	for name, inst := range e.actors {
		instances = append(instances, inst)
		delete(e.actors, name)
	}
	e.mu.Unlock()

	for _, inst := range instances {
		inst.ctx.mailbox.Stop()
	}

	for _, inst := range instances {
		select {
		case <-inst.ref.done:
		case <-ctx.Done():
			return &ShutdownTimeoutError{RunningCount: e.runningCount(instances)}
		}
	}
	return nil
}

func (e *Engine) wrapReceive(a Actor) ReceiveFunc {
	receive := ReceiveFunc(a.Receive)
	for i := len(e.middleware) - 1; i >= 0; i-- {
		receive = e.middleware[i](receive)
	}
	return receive
}

func (e *Engine) newMailbox() Mailbox {
	if e.mailboxFactory != nil {
		return e.mailboxFactory()
	}
	return NewChanMailboxWithPolicy(e.mailboxSize, e.mailboxPolicy)
}

func (e *Engine) newDeliver(name string, mailbox Mailbox) deliverFunc {
	return func(env Envelope) error {
		result := mailbox.Push(env)
		if result.Dropped != nil {
			e.metrics.MessageDropped(name, result.QueueDepth, string(result.Dropped.Reason))
			e.emitDeadLetter(name, *result.Dropped)
		}
		if result.Accepted {
			e.metrics.MessageQueued(name, result.QueueDepth)
			return nil
		}

		if result.Reason == MailboxDropReasonFull {
			return &MailboxFullError{Name: name}
		}
		if result.Reason != MailboxDropReasonStopped {
			return &MailboxRejectedError{Name: name, Reason: result.Reason}
		}
		return &ActorStoppedError{Name: name}
	}
}

func (e *Engine) emitDeadLetter(target string, dropped MailboxDrop) {
	if e.deadLetterHandler == nil {
		return
	}
	e.deadLetterHandler(DeadLetter{
		Target:   target,
		Envelope: dropped.Envelope,
		Reason:   dropped.Reason,
	})
}

func resolveActorFactory(a Actor) ActorFactory {
	factory, ok := a.(ActorFactory)
	if !ok {
		return nil
	}
	return factory
}

func (inst *actorInstance) restart(e *Engine) bool {
	if inst.actorFactory == nil || !inst.allowRestart(e.supervision) {
		return false
	}

	newActor := inst.actorFactory.NewActor()
	if newActor == nil {
		return false
	}

	e.metrics.ActorStateChanged(inst.ref.name, ActorStateRestarting)
	inst.triggerOnStop()
	if backoff := e.supervision.RestartBackoff; backoff > 0 {
		time.Sleep(backoff)
	}

	inst.actor = newActor
	inst.receive = e.wrapReceive(newActor)
	inst.ctx = &actorContext{
		self:    inst.ref,
		engine:  e,
		mailbox: inst.ctx.mailbox,
	}
	e.metrics.ActorRestarted(inst.ref.name)
	inst.triggerOnStart()
	e.metrics.ActorStateChanged(inst.ref.name, ActorStateRunning)
	return true
}

func (inst *actorInstance) allowRestart(cfg SupervisionConfig) bool {
	now := time.Now()
	if cfg.MaxRestarts <= 0 {
		inst.restartHistory = append(inst.restartHistory, now)
		return true
	}

	if cfg.RestartWindow > 0 {
		cutoff := now.Add(-cfg.RestartWindow)
		filtered := inst.restartHistory[:0]
		for _, ts := range inst.restartHistory {
			if ts.After(cutoff) {
				filtered = append(filtered, ts)
			}
		}
		inst.restartHistory = filtered
	}

	if len(inst.restartHistory) >= cfg.MaxRestarts {
		return false
	}

	inst.restartHistory = append(inst.restartHistory, now)
	return true
}

func (e *Engine) runningCount(instances []*actorInstance) int {
	count := 0
	for _, inst := range instances {
		select {
		case <-inst.ref.done:
		default:
			count++
		}
	}
	return count
}

func mailboxPolicyFromBackpressure(policy BackpressurePolicy) MailboxPolicy {
	switch policy {
	case BackpressureDropNewest:
		return NewDropNewestMailboxPolicy()
	case BackpressureDropOldest:
		return NewDropOldestMailboxPolicy()
	default:
		return NewBlockingMailboxPolicy()
	}
}
