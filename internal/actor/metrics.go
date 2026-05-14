package actor

import (
	"sync"
	"sync/atomic"
	"time"
)

// ActorState 描述 actor 当前所处的生命周期状态
type ActorState string

const (
	ActorStateUnknown    ActorState = "unknown"
	ActorStateStarting   ActorState = "starting"
	ActorStateRunning    ActorState = "running"
	ActorStateRestarting ActorState = "restarting"
	ActorStateStopping   ActorState = "stopping"
	ActorStateStopped    ActorState = "stopped"
)

// MetricEventType 描述最近事件的类型
type MetricEventType string

const (
	MetricEventActorStateChanged MetricEventType = "actor_state_changed"
	MetricEventMessageQueued     MetricEventType = "message_queued"
	MetricEventMessageDropped    MetricEventType = "message_dropped"
	MetricEventMessageProcessed  MetricEventType = "message_processed"
	MetricEventPanicRecovered    MetricEventType = "panic_recovered"
	MetricEventWatchAdded        MetricEventType = "watch_added"
	MetricEventWatchRemoved      MetricEventType = "watch_removed"
	MetricEventTimerScheduled    MetricEventType = "timer_scheduled"
	MetricEventTimerCanceled     MetricEventType = "timer_canceled"
	MetricEventTimerFired        MetricEventType = "timer_fired"
	MetricEventActorRestarted    MetricEventType = "actor_restarted"
)

// MetricEvent 记录最近一条关键事件，便于诊断生命周期与背压问题
type MetricEvent struct {
	Time         time.Time
	Type         MetricEventType
	Actor        string
	RelatedActor string
	Value        int64
	Detail       string
}

// Metrics 定义 actor 系统的指标收集接口
type Metrics interface {
	ActorSpawned(name string)
	ActorStopped(name string)
	ActorRestarted(name string)
	ActorStateChanged(name string, state ActorState)
	MessageDequeued(name string, queueDepth int)
	MessageProcessed(name string, duration time.Duration)
	MessageQueued(name string, queueDepth int)
	MessageDropped(name string, queueDepth int, reason string)
	PanicRecovered(name string)
	WatchAdded(watcher, watched string)
	WatchRemoved(watcher, watched string)
	TimerScheduled(name string)
	TimerCanceled(name string)
	TimerFired(name string)
	GetSnapshot() MetricsSnapshot
}

// MetricsSnapshot 是某一时刻的指标快照
type MetricsSnapshot struct {
	TotalActors   int64
	TotalMessages int64
	TotalDropped  int64
	TotalPanics   int64
	TotalRestarts int64
	TotalWatches  int64
	RecentEvents  []MetricEvent
	ActorStats    map[string]ActorStats
}

// ActorStats 是单个 actor 的运行统计信息
type ActorStats struct {
	Name                string
	State               ActorState
	MessagesQueued      int64
	MessagesDropped     int64
	MessagesProcessed   int64
	Panics              int64
	Restarts            int64
	ActiveWatches       int64
	QueueDepth          int64
	MaxQueueDepth       int64
	LastProcessingNanos int64
	TimersScheduled     int64
	TimersCanceled      int64
	TimersFired         int64
}

type defaultMetrics struct {
	totalActors   atomic.Int64
	totalMessages atomic.Int64
	totalDropped  atomic.Int64
	totalPanics   atomic.Int64
	totalRestarts atomic.Int64
	totalWatches  atomic.Int64

	mu           sync.RWMutex
	actorStats   map[string]*actorStatsEntry
	recentEvents []MetricEvent
}

type actorStatsEntry struct {
	state             atomic.Value
	messagesQueued    atomic.Int64
	messagesDropped   atomic.Int64
	messagesProcessed atomic.Int64
	panics            atomic.Int64
	restarts          atomic.Int64
	activeWatches     atomic.Int64
	queueDepth        atomic.Int64
	maxQueueDepth     atomic.Int64
	lastProcessNanos  atomic.Int64
	timersScheduled   atomic.Int64
	timersCanceled    atomic.Int64
	timersFired       atomic.Int64
}

// NewDefaultMetrics 创建默认的指标收集器
func NewDefaultMetrics() Metrics {
	return &defaultMetrics{
		actorStats: make(map[string]*actorStatsEntry),
	}
}

func (m *defaultMetrics) getOrCreateEntry(name string) *actorStatsEntry {
	m.mu.RLock()
	entry, ok := m.actorStats[name]
	m.mu.RUnlock()
	if ok {
		return entry
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok = m.actorStats[name]
	if ok {
		return entry
	}
	entry = &actorStatsEntry{}
	entry.state.Store(ActorStateUnknown)
	m.actorStats[name] = entry
	return entry
}

func (m *defaultMetrics) ActorSpawned(name string) {
	m.totalActors.Add(1)
	m.getOrCreateEntry(name)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventActorStateChanged,
		Actor: name,
		Detail: string(ActorStateStarting),
	})
}

func (m *defaultMetrics) ActorStopped(name string) {
	m.totalActors.Add(-1)
}

func (m *defaultMetrics) ActorRestarted(name string) {
	m.totalRestarts.Add(1)
	entry := m.getOrCreateEntry(name)
	entry.restarts.Add(1)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventActorRestarted,
		Actor: name,
	})
}

func (m *defaultMetrics) ActorStateChanged(name string, state ActorState) {
	entry := m.getOrCreateEntry(name)
	entry.state.Store(state)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventActorStateChanged,
		Actor: name,
		Detail: string(state),
	})
}

func (m *defaultMetrics) MessageProcessed(name string, duration time.Duration) {
	m.totalMessages.Add(1)
	entry := m.getOrCreateEntry(name)
	entry.messagesProcessed.Add(1)
	entry.lastProcessNanos.Store(duration.Nanoseconds())
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventMessageProcessed,
		Actor: name,
		Value: duration.Nanoseconds(),
	})
}

func (m *defaultMetrics) MessageQueued(name string, queueDepth int) {
	entry := m.getOrCreateEntry(name)
	entry.messagesQueued.Add(1)
	m.setQueueDepth(entry, queueDepth)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventMessageQueued,
		Actor: name,
		Value: int64(queueDepth),
	})
}

func (m *defaultMetrics) MessageDequeued(name string, queueDepth int) {
	entry := m.getOrCreateEntry(name)
	m.setQueueDepth(entry, queueDepth)
}

func (m *defaultMetrics) MessageDropped(name string, queueDepth int, reason string) {
	m.totalDropped.Add(1)
	entry := m.getOrCreateEntry(name)
	entry.messagesDropped.Add(1)
	m.setQueueDepth(entry, queueDepth)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventMessageDropped,
		Actor: name,
		Value: int64(queueDepth),
		Detail: reason,
	})
}

func (m *defaultMetrics) PanicRecovered(name string) {
	m.totalPanics.Add(1)
	entry := m.getOrCreateEntry(name)
	entry.panics.Add(1)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventPanicRecovered,
		Actor: name,
	})
}

func (m *defaultMetrics) WatchAdded(watcher, watched string) {
	m.totalWatches.Add(1)
	entry := m.getOrCreateEntry(watcher)
	entry.activeWatches.Add(1)
	m.recordEvent(MetricEvent{
		Time:         time.Now(),
		Type:         MetricEventWatchAdded,
		Actor:        watcher,
		RelatedActor: watched,
	})
}

func (m *defaultMetrics) WatchRemoved(watcher, watched string) {
	m.totalWatches.Add(-1)
	entry := m.getOrCreateEntry(watcher)
	entry.activeWatches.Add(-1)
	m.recordEvent(MetricEvent{
		Time:         time.Now(),
		Type:         MetricEventWatchRemoved,
		Actor:        watcher,
		RelatedActor: watched,
	})
}

func (m *defaultMetrics) TimerScheduled(name string) {
	entry := m.getOrCreateEntry(name)
	entry.timersScheduled.Add(1)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventTimerScheduled,
		Actor: name,
	})
}

func (m *defaultMetrics) TimerCanceled(name string) {
	entry := m.getOrCreateEntry(name)
	entry.timersCanceled.Add(1)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventTimerCanceled,
		Actor: name,
	})
}

func (m *defaultMetrics) TimerFired(name string) {
	entry := m.getOrCreateEntry(name)
	entry.timersFired.Add(1)
	m.recordEvent(MetricEvent{
		Time:  time.Now(),
		Type:  MetricEventTimerFired,
		Actor: name,
	})
}

func (m *defaultMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	actorStats := make(map[string]ActorStats, len(m.actorStats))
	for name, entry := range m.actorStats {
		state := ActorStateUnknown
		if raw := entry.state.Load(); raw != nil {
			state, _ = raw.(ActorState)
		}
		actorStats[name] = ActorStats{
			Name:                name,
			State:               state,
			MessagesQueued:      entry.messagesQueued.Load(),
			MessagesDropped:     entry.messagesDropped.Load(),
			MessagesProcessed:   entry.messagesProcessed.Load(),
			Panics:              entry.panics.Load(),
			Restarts:            entry.restarts.Load(),
			ActiveWatches:       entry.activeWatches.Load(),
			QueueDepth:          entry.queueDepth.Load(),
			MaxQueueDepth:       entry.maxQueueDepth.Load(),
			LastProcessingNanos: entry.lastProcessNanos.Load(),
			TimersScheduled:     entry.timersScheduled.Load(),
			TimersCanceled:      entry.timersCanceled.Load(),
			TimersFired:         entry.timersFired.Load(),
		}
	}

	return MetricsSnapshot{
		TotalActors:   m.totalActors.Load(),
		TotalMessages: m.totalMessages.Load(),
		TotalDropped:  m.totalDropped.Load(),
		TotalPanics:   m.totalPanics.Load(),
		TotalRestarts: m.totalRestarts.Load(),
		TotalWatches:  m.totalWatches.Load(),
		RecentEvents:  append([]MetricEvent(nil), m.recentEvents...),
		ActorStats:    actorStats,
	}
}

type noopMetrics struct{}

func (noopMetrics) ActorSpawned(string)                             {}
func (noopMetrics) ActorStopped(string)                             {}
func (noopMetrics) ActorRestarted(string)                           {}
func (noopMetrics) ActorStateChanged(string, ActorState)            {}
func (noopMetrics) MessageDequeued(string, int)                     {}
func (noopMetrics) MessageProcessed(string, time.Duration)          {}
func (noopMetrics) MessageQueued(string, int)                       {}
func (noopMetrics) MessageDropped(string, int, string)              {}
func (noopMetrics) PanicRecovered(string)                           {}
func (noopMetrics) WatchAdded(_, _ string)                          {}
func (noopMetrics) WatchRemoved(_, _ string)                        {}
func (noopMetrics) TimerScheduled(string)                           {}
func (noopMetrics) TimerCanceled(string)                            {}
func (noopMetrics) TimerFired(string)                               {}
func (noopMetrics) GetSnapshot() MetricsSnapshot {
	return MetricsSnapshot{ActorStats: make(map[string]ActorStats)}
}

func (m *defaultMetrics) setQueueDepth(entry *actorStatsEntry, queueDepth int) {
	depth := int64(queueDepth)
	entry.queueDepth.Store(depth)
	for {
		current := entry.maxQueueDepth.Load()
		if depth <= current {
			return
		}
		if entry.maxQueueDepth.CompareAndSwap(current, depth) {
			return
		}
	}
}

func (m *defaultMetrics) recordEvent(event MetricEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recentEvents = append(m.recentEvents, event)
	if len(m.recentEvents) > 256 {
		m.recentEvents = append([]MetricEvent(nil), m.recentEvents[len(m.recentEvents)-256:]...)
	}
}
