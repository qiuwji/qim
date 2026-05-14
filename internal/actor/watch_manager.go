package actor

import "sync"

// watchedState 记录一个被监听 actor 的所有监听者和生命周期控制
type watchedState struct {
	watchers map[*ActorRef]struct{}
	stopCh   chan struct{}
}

// watchManager 管理 actor 之间的监听关系和终止信号分发
type watchManager struct {
	mu      sync.RWMutex
	watches map[*ActorRef]*watchedState
	metrics Metrics
}

func newWatchManager() *watchManager {
	return &watchManager{
		watches: make(map[*ActorRef]*watchedState),
		metrics: noopMetrics{},
	}
}

func (wm *watchManager) setMetrics(metrics Metrics) {
	if metrics == nil {
		wm.metrics = noopMetrics{}
		return
	}
	wm.metrics = metrics
}

func (wm *watchManager) add(watcher, watched *ActorRef) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	select {
	case <-watched.Done():
		watcher.Tell(Terminated{Who: watched})
		return false
	default:
	}

	state, ok := wm.watches[watched]
	if !ok {
		state = &watchedState{
			watchers: make(map[*ActorRef]struct{}),
			stopCh:   make(chan struct{}),
		}
		wm.watches[watched] = state
		go wm.monitor(watched, state)
	}

	if _, exists := state.watchers[watcher]; exists {
		return false
	}

	state.watchers[watcher] = struct{}{}
	return true
}

func (wm *watchManager) remove(watcher, watched *ActorRef) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	state, ok := wm.watches[watched]
	if !ok {
		return false
	}

	if _, exists := state.watchers[watcher]; !exists {
		return false
	}

	delete(state.watchers, watcher)

	if len(state.watchers) == 0 {
		close(state.stopCh)
		delete(wm.watches, watched)
	}

	return true
}

func (wm *watchManager) cleanupForWatcher(watcher *ActorRef) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	var toRemove []*ActorRef
	var toClose []chan struct{}
	var removedWatches []*ActorRef
	for watched, state := range wm.watches {
		if _, exists := state.watchers[watcher]; exists {
			delete(state.watchers, watcher)
			removedWatches = append(removedWatches, watched)
			if len(state.watchers) == 0 {
				toRemove = append(toRemove, watched)
				toClose = append(toClose, state.stopCh)
			}
		}
	}
	for i, watched := range toRemove {
		close(toClose[i])
		delete(wm.watches, watched)
	}
	for _, watched := range removedWatches {
		wm.metrics.WatchRemoved(watcher.name, watched.name)
	}
}

func (wm *watchManager) monitor(watched *ActorRef, state *watchedState) {
	select {
	case <-watched.Done():
	case <-state.stopCh:
		return
	}

	wm.mu.Lock()
	if wm.watches[watched] != state {
		wm.mu.Unlock()
		return
	}
	watchers := make([]*ActorRef, 0, len(state.watchers))
	for watcher := range state.watchers {
		watchers = append(watchers, watcher)
	}
	delete(wm.watches, watched)
	wm.mu.Unlock()

	for _, watcher := range watchers {
		wm.metrics.WatchRemoved(watcher.name, watched.name)
		watcher.Tell(Terminated{Who: watched})
	}
}
