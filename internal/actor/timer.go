package actor

import (
	"sync"
	"time"
)

// Timer 是可取消的定时器，由 ScheduleAfter 返回
type Timer struct {
	cancel func() bool
}

// Cancel 取消定时器，阻止消息发送
func (t *Timer) Cancel() {
	if t == nil || t.cancel == nil {
		return
	}
	t.cancel()
}

type timerManager struct {
	mu      sync.Mutex
	timers  map[*ActorRef]map[*managedTimer]struct{}
	metrics Metrics
}

type managedTimer struct {
	owner   *ActorRef
	timer   *time.Timer
	manager *timerManager
	once    sync.Once
}

func newTimerManager(metrics Metrics) *timerManager {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &timerManager{
		timers:  make(map[*ActorRef]map[*managedTimer]struct{}),
		metrics: metrics,
	}
}

func (tm *timerManager) setMetrics(metrics Metrics) {
	if metrics == nil {
		tm.metrics = noopMetrics{}
		return
	}
	tm.metrics = metrics
}

func (tm *timerManager) Schedule(owner *ActorRef, d time.Duration, msg any) *Timer {
	mt := &managedTimer{
		owner:   owner,
		manager: tm,
	}

	tm.register(mt)
	tm.metrics.TimerScheduled(owner.name)

	mt.timer = time.AfterFunc(d, func() {
		if !mt.finish() {
			return
		}
		tm.metrics.TimerFired(owner.name)
		_ = owner.Tell(msg)
	})

	return &Timer{
		cancel: func() bool {
			if mt.timer == nil || !mt.timer.Stop() {
				return false
			}
			if !mt.finish() {
				return false
			}
			tm.metrics.TimerCanceled(owner.name)
			return true
		},
	}
}

func (tm *timerManager) cancelOwner(owner *ActorRef) {
	tm.mu.Lock()
	ownerTimers := tm.timers[owner]
	delete(tm.timers, owner)
	tm.mu.Unlock()

	for mt := range ownerTimers {
		if mt.timer == nil || !mt.timer.Stop() {
			continue
		}
		if mt.finish() {
			tm.metrics.TimerCanceled(owner.name)
		}
	}
}

func (tm *timerManager) register(mt *managedTimer) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.timers[mt.owner] == nil {
		tm.timers[mt.owner] = make(map[*managedTimer]struct{})
	}
	tm.timers[mt.owner][mt] = struct{}{}
}

func (tm *timerManager) unregister(mt *managedTimer) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ownerTimers, ok := tm.timers[mt.owner]
	if !ok {
		return
	}
	delete(ownerTimers, mt)
	if len(ownerTimers) == 0 {
		delete(tm.timers, mt.owner)
	}
}

func (mt *managedTimer) finish() bool {
	finished := false
	mt.once.Do(func() {
		finished = true
		mt.manager.unregister(mt)
	})
	return finished
}
