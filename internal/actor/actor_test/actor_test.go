package actor_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"qim/internal/actor"
)

type pingMsg struct{}

type pongMsg struct{}

type PingActor struct {
	pongCount atomic.Int64
}

func (p *PingActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case pingMsg:
		ctx.Reply(pongMsg{})
	case pongMsg:
		p.pongCount.Add(1)
	}
}

type actorFunc func(ctx actor.Context)

func (f actorFunc) Receive(ctx actor.Context) { f(ctx) }

type lifecycleFuncs struct {
	onStart func(ctx actor.Context)
	onStop  func(ctx actor.Context)
}

func (l lifecycleFuncs) OnStart(ctx actor.Context) { l.onStart(ctx) }
func (l lifecycleFuncs) OnStop(ctx actor.Context)  { l.onStop(ctx) }

type lifecycleActor struct {
	actorFunc
	lifecycleFuncs
}

type restartablePanicActor struct {
	started *atomic.Int32
	stopped *atomic.Int32
	safe    *atomic.Int32
}

func (a *restartablePanicActor) Receive(ctx actor.Context) {
	switch ctx.Message() {
	case "boom":
		panic("restartable panic")
	case "safe":
		a.safe.Add(1)
	}
}

func (a *restartablePanicActor) OnStart(ctx actor.Context) {
	a.started.Add(1)
}

func (a *restartablePanicActor) OnStop(ctx actor.Context) {
	a.stopped.Add(1)
}

func (a *restartablePanicActor) NewActor() actor.Actor {
	return &restartablePanicActor{
		started: a.started,
		stopped: a.stopped,
		safe:    a.safe,
	}
}

func TestTell(t *testing.T) {
	engine := actor.NewEngine()
	ref, err := engine.Spawn("test", &PingActor{})
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	if err := ref.Tell(pingMsg{}); err != nil {
		t.Fatalf("tell failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
}

func TestTellFrom(t *testing.T) {
	var received atomic.Bool

	engine := actor.NewEngine()
	sender, _ := engine.Spawn("sender", actorFunc(func(ctx actor.Context) {}))
	receiver, _ := engine.Spawn("receiver", actorFunc(func(ctx actor.Context) {
		if ctx.Sender() != nil && ctx.Sender().Name() == "sender" {
			received.Store(true)
		}
	}))

	receiver.TellFrom(sender, pingMsg{})
	time.Sleep(100 * time.Millisecond)

	if !received.Load() {
		t.Fatal("receiver should see sender via TellFrom")
	}
}

func TestAsk(t *testing.T) {
	engine := actor.NewEngine()
	ref, err := engine.Spawn("test", &PingActor{})
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	result, err := ref.Ask(pingMsg{}, 3*time.Second)
	if err != nil {
		t.Fatalf("ask failed: %v", err)
	}
	if _, ok := result.(pongMsg); !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
}

func TestAskTimeout(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", &PingActor{})

	_, err := ref.Ask("no-reply", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when actor does not reply")
	}
}

func TestAskImmediateMode(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", &PingActor{})

	_, err := ref.Ask("no-reply", 0)
	if err == nil {
		t.Fatal("immediate ask with no reply should fail")
	}
}

func TestConcurrentAsk(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", &PingActor{})

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := ref.Ask(pingMsg{}, 3*time.Second)
			if err == nil {
				if _, ok := result.(pongMsg); ok {
					successCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	if successCount.Load() != 50 {
		t.Fatalf("expected 50 successful asks, got %d", successCount.Load())
	}
}

func TestDuplicateSpawn(t *testing.T) {
	engine := actor.NewEngine()
	_, err := engine.Spawn("test", &PingActor{})
	if err != nil {
		t.Fatalf("first spawn failed: %v", err)
	}

	_, err = engine.Spawn("test", &PingActor{})
	if err != actor.ErrActorExists {
		t.Fatalf("expected ErrActorExists, got: %v", err)
	}
}

func TestStop(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", &PingActor{})

	engine.Stop("test")
	<-ref.Done()

	if err := ref.Tell(pingMsg{}); err == nil {
		t.Fatal("expected error after stop")
	}
}

func TestStopNonExistent(t *testing.T) {
	engine := actor.NewEngine()
	engine.Stop("nonexistent")
}

func TestLookup(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", &PingActor{})

	found, ok := engine.Lookup("test")
	if !ok {
		t.Fatal("expected to find actor")
	}
	if found.Name() != ref.Name() {
		t.Fatalf("expected name %s, got %s", ref.Name(), found.Name())
	}

	_, ok = engine.Lookup("nonexistent")
	if ok {
		t.Fatal("expected not to find actor")
	}
}

func TestLookupAfterStop(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", &PingActor{})

	engine.Stop("test")
	<-ref.Done()

	_, ok := engine.Lookup("test")
	if ok {
		t.Fatal("expected not to find stopped actor")
	}
}

func TestLifecycle(t *testing.T) {
	var started, stopped atomic.Bool

	engine := actor.NewEngine()
	ref, _ := engine.Spawn("lc", &lifecycleActor{
		actorFunc: actorFunc(func(ctx actor.Context) {}),
		lifecycleFuncs: lifecycleFuncs{
			onStart: func(ctx actor.Context) { started.Store(true) },
			onStop:  func(ctx actor.Context) { stopped.Store(true) },
		},
	})

	time.Sleep(50 * time.Millisecond)
	if !started.Load() {
		t.Fatal("OnStart not called")
	}

	engine.Stop("lc")
	<-ref.Done()

	if !stopped.Load() {
		t.Fatal("OnStop not called")
	}
}

func TestWatch(t *testing.T) {
	var terminated atomic.Bool

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if _, ok := ctx.Message().(string); ok && ctx.Message() == "start" {
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated.Store(true)
		}
	}))

	time.Sleep(50 * time.Millisecond)

	watcher.Tell("start")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if !terminated.Load() {
		t.Fatal("watcher did not receive Terminated")
	}
}

func TestWatchAlreadyStopped(t *testing.T) {
	var terminated atomic.Bool

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	engine.Stop("watched")
	<-watched.Done()

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if _, ok := ctx.Message().(string); ok && ctx.Message() == "start" {
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated.Store(true)
		}
	}))

	time.Sleep(50 * time.Millisecond)
	watcher.Tell("start")
	time.Sleep(100 * time.Millisecond)

	if !terminated.Load() {
		t.Fatal("watcher should receive Terminated immediately when watching already-stopped actor")
	}
}

func TestWatchIdempotent(t *testing.T) {
	var terminatedCount atomic.Int32

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if _, ok := ctx.Message().(string); ok && ctx.Message() == "watch" {
			ctx.Watch(watched)
			ctx.Watch(watched)
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminatedCount.Add(1)
		}
	}))

	time.Sleep(50 * time.Millisecond)
	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if terminatedCount.Load() != 1 {
		t.Fatalf("expected exactly 1 Terminated, got %d", terminatedCount.Load())
	}
}

func TestUnwatch(t *testing.T) {
	var terminated atomic.Int32

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		switch ctx.Message() {
		case "watch":
			ctx.Watch(watched)
		case "unwatch":
			ctx.Unwatch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated.Add(1)
		}
	}))

	time.Sleep(50 * time.Millisecond)

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	watcher.Tell("unwatch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if terminated.Load() != 0 {
		t.Fatal("watcher should not receive Terminated after unwatch")
	}
}

func TestScheduleAfter(t *testing.T) {
	var received atomic.Bool

	engine := actor.NewEngine()
	ref, _ := engine.Spawn("timer", actorFunc(func(ctx actor.Context) {
		switch ctx.Message() {
		case "tick":
			received.Store(true)
		case "start":
			ctx.ScheduleAfter(50*time.Millisecond, "tick")
		}
	}))

	ref.Tell("start")
	time.Sleep(200 * time.Millisecond)

	if !received.Load() {
		t.Fatal("scheduled message not received")
	}
}

func TestScheduleAfterCancel(t *testing.T) {
	var received atomic.Int32

	engine := actor.NewEngine()
	ref, _ := engine.Spawn("timer", actorFunc(func(ctx actor.Context) {
		switch msg := ctx.Message().(type) {
		case string:
			switch msg {
			case "start":
				timer := ctx.ScheduleAfter(100*time.Millisecond, "tick")
				timer.Cancel()
			case "tick":
				received.Add(1)
			}
		}
	}))

	ref.Tell("start")
	time.Sleep(200 * time.Millisecond)

	if received.Load() != 0 {
		t.Fatal("cancelled timer should not fire")
	}
}

func TestScheduleAfterActorStopped(t *testing.T) {
	var received atomic.Int32

	engine := actor.NewEngine()
	ref, _ := engine.Spawn("timer", actorFunc(func(ctx actor.Context) {
		switch ctx.Message() {
		case "start":
			ctx.ScheduleAfter(200*time.Millisecond, "tick")
		case "tick":
			received.Add(1)
		}
	}))

	ref.Tell("start")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("timer")
	<-ref.Done()

	time.Sleep(300 * time.Millisecond)

	if received.Load() != 0 {
		t.Fatal("scheduled message should be silently dropped after actor stops")
	}
}

func TestScheduleAfterManagedByEngine(t *testing.T) {
	var received atomic.Int32

	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))
	ref, _ := engine.Spawn("timer-owner", actorFunc(func(ctx actor.Context) {
		switch ctx.Message() {
		case "start":
			ctx.ScheduleAfter(200*time.Millisecond, "tick-1")
			ctx.ScheduleAfter(200*time.Millisecond, "tick-2")
		case "tick-1", "tick-2":
			received.Add(1)
		}
	}))

	ref.Tell("start")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("timer-owner")
	<-ref.Done()
	time.Sleep(250 * time.Millisecond)

	if received.Load() != 0 {
		t.Fatalf("expected managed timers to be cancelled on actor stop, got %d ticks", received.Load())
	}

	stats, ok := metrics.GetSnapshot().ActorStats["timer-owner"]
	if !ok {
		t.Fatal("expected timer-owner stats")
	}
	if stats.TimersScheduled != 2 {
		t.Fatalf("expected 2 scheduled timers, got %d", stats.TimersScheduled)
	}
	if stats.TimersCanceled != 2 {
		t.Fatalf("expected 2 cancelled timers, got %d", stats.TimersCanceled)
	}
}

func TestMiddleware(t *testing.T) {
	var mu sync.Mutex
	var order []string

	mw1 := func(next actor.ReceiveFunc) actor.ReceiveFunc {
		return func(ctx actor.Context) {
			mu.Lock()
			order = append(order, "mw1-before")
			mu.Unlock()
			next(ctx)
			mu.Lock()
			order = append(order, "mw1-after")
			mu.Unlock()
		}
	}

	mw2 := func(next actor.ReceiveFunc) actor.ReceiveFunc {
		return func(ctx actor.Context) {
			mu.Lock()
			order = append(order, "mw2-before")
			mu.Unlock()
			next(ctx)
			mu.Lock()
			order = append(order, "mw2-after")
			mu.Unlock()
		}
	}

	engine := actor.NewEngine(actor.WithMiddleware(mw1, mw2))
	_, _ = engine.Spawn("test", actorFunc(func(ctx actor.Context) {
		mu.Lock()
		order = append(order, "receive")
		mu.Unlock()
	}))

	ref, _ := engine.Lookup("test")
	ref.Tell("hello")
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	expected := []string{"mw1-before", "mw2-before", "receive", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("at index %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestPanicRecovery(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("panicker", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "boom" {
			panic("test panic")
		}
	}))

	ref.Tell("boom")
	time.Sleep(50 * time.Millisecond)

	if err := ref.Tell(pingMsg{}); err != nil {
		t.Fatal("actor should still be alive after panic recovery")
	}
}

func TestPoisonPill(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))

	ref.Tell(actor.PoisonPill{})
	<-ref.Done()

	engine.Stop("test")
	if err := ref.Tell(pingMsg{}); err == nil {
		t.Fatal("expected error after stop")
	}
}

func TestPoisonPillRemovesActorFromRegistry(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))

	ref.Tell(actor.PoisonPill{})
	<-ref.Done()

	if _, ok := engine.Lookup("test"); ok {
		t.Fatal("poison pill should remove actor from registry")
	}

	respawned, err := engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))
	if err != nil {
		t.Fatalf("expected respawn after poison pill to succeed: %v", err)
	}
	engine.Stop(respawned.Name())
	<-respawned.Done()
}

func TestReplyTwice(t *testing.T) {
	engine := actor.NewEngine()
	_, _ = engine.Spawn("test", actorFunc(func(ctx actor.Context) {
		ctx.Reply(pongMsg{})
		err := ctx.Reply(pongMsg{})
		if err == nil {
			panic("second Reply should return error")
		}
	}))

	ref, _ := engine.Lookup("test")
	_, err := ref.Ask(pingMsg{}, 3*time.Second)
	if err != nil {
		t.Fatalf("first ask failed: %v", err)
	}
}

func TestAskNoReplyAutoReplies(t *testing.T) {
	engine := actor.NewEngine()
	_, _ = engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))

	ref, _ := engine.Lookup("test")
	result, err := ref.Ask("no-reply-msg", 3*time.Second)
	if err == nil {
		t.Fatalf("expected error when actor does not reply, got result: %v", result)
	}
	if err.Error() != actor.ErrNoReply.Error() {
		t.Fatalf("expected ErrNoReply, got: %v", err)
	}
}

func TestAskNoReplyOnPanic(t *testing.T) {
	engine := actor.NewEngine(actor.WithSupervisionStrategy(actor.StrategyResume))
	_, _ = engine.Spawn("test", actorFunc(func(ctx actor.Context) {
		panic("boom")
	}))

	ref, _ := engine.Lookup("test")
	result, err := ref.Ask("trigger-panic", 3*time.Second)
	if err == nil {
		t.Fatalf("expected error when actor panics without reply, got result: %v", result)
	}
	if err.Error() != actor.ErrNoReply.Error() {
		t.Fatalf("expected ErrNoReply, got: %v", err)
	}
}

func TestConcurrentTellAndStop(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", actorFunc(func(ctx actor.Context) {
		time.Sleep(10 * time.Millisecond)
	}))

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			if err := ref.Tell(pingMsg{}); err != nil {
				return
			}
		}
	}()

	time.Sleep(20 * time.Millisecond)
	engine.Stop("test")
	<-ref.Done()
	<-done
}

func TestMailboxDrainOnStop(t *testing.T) {
	var processed atomic.Int32

	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", actorFunc(func(ctx actor.Context) {
		processed.Add(1)
	}))

	for i := 0; i < 5; i++ {
		ref.Tell(pingMsg{})
	}
	time.Sleep(50 * time.Millisecond)

	engine.Stop("test")
	<-ref.Done()

	if processed.Load() != 5 {
		t.Fatalf("expected 5 messages processed, got %d", processed.Load())
	}
}

func TestShutdown(t *testing.T) {
	var started, stopped atomic.Int32

	engine := actor.NewEngine()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("actor-%d", i)
		_, _ = engine.Spawn(name, &lifecycleActor{
			actorFunc: actorFunc(func(ctx actor.Context) {}),
			lifecycleFuncs: lifecycleFuncs{
				onStart: func(ctx actor.Context) { started.Add(1) },
				onStop:  func(ctx actor.Context) { stopped.Add(1) },
			},
		})
	}

	time.Sleep(50 * time.Millisecond)
	if started.Load() != 10 {
		t.Fatalf("expected 10 started, got %d", started.Load())
	}

	err := engine.Shutdown()
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if stopped.Load() != 10 {
		t.Fatalf("expected 10 stopped, got %d", stopped.Load())
	}

	_, ok := engine.Lookup("actor-0")
	if ok {
		t.Fatal("expected all actors to be removed after shutdown")
	}
}

func TestShutdownWithTimeout(t *testing.T) {
	engine := actor.NewEngine()
	slowStarted := make(chan struct{})
	ref, _ := engine.Spawn("slow", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "block" {
			close(slowStarted)
			time.Sleep(5 * time.Second)
		}
	}))

	ref.Tell("block")
	<-slowStarted

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := engine.ShutdownWithContext(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestShutdownEmpty(t *testing.T) {
	engine := actor.NewEngine()
	err := engine.Shutdown()
	if err != nil {
		t.Fatalf("shutdown empty engine should succeed: %v", err)
	}
}

func TestShutdownIdempotent(t *testing.T) {
	engine := actor.NewEngine()
	_, _ = engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))

	err := engine.Shutdown()
	if err != nil {
		t.Fatalf("first shutdown failed: %v", err)
	}

	err = engine.Shutdown()
	if err != nil {
		t.Fatalf("second shutdown failed: %v", err)
	}
}

func TestMetrics(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))

	ref, _ := engine.Spawn("test-actor", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "panic" {
			panic("test panic")
		}
	}))

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalActors != 1 {
		t.Fatalf("expected 1 actor, got %d", snapshot.TotalActors)
	}

	for i := 0; i < 10; i++ {
		ref.Tell(pingMsg{})
	}
	time.Sleep(100 * time.Millisecond)

	snapshot = metrics.GetSnapshot()
	if snapshot.TotalMessages < 10 {
		t.Fatalf("expected at least 10 messages, got %d", snapshot.TotalMessages)
	}

	ref.Tell("panic")
	time.Sleep(100 * time.Millisecond)

	snapshot = metrics.GetSnapshot()
	if snapshot.TotalPanics != 1 {
		t.Fatalf("expected 1 panic, got %d", snapshot.TotalPanics)
	}

	engine.Stop("test-actor")
	<-ref.Done()

	snapshot = metrics.GetSnapshot()
	if snapshot.TotalActors != 0 {
		t.Fatalf("expected 0 actors after stop, got %d", snapshot.TotalActors)
	}
}

func TestMetricsWatch(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))

	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))
	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
		if ctx.Message() == "unwatch" {
			ctx.Unwatch(watched)
		}
	}))

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalWatches != 1 {
		t.Fatalf("expected 1 watch, got %d", snapshot.TotalWatches)
	}

	watcher.Tell("unwatch")
	time.Sleep(50 * time.Millisecond)

	snapshot = metrics.GetSnapshot()
	if snapshot.TotalWatches != 0 {
		t.Fatalf("expected 0 watches after unwatch, got %d", snapshot.TotalWatches)
	}
}

func TestMetricsWatchCleanupOnWatchedStop(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))

	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))
	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
	}))

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	<-watched.Done()
	time.Sleep(50 * time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalWatches != 0 {
		t.Fatalf("expected 0 watches after watched stop, got %d", snapshot.TotalWatches)
	}

	stats, ok := snapshot.ActorStats["watcher"]
	if !ok {
		t.Fatal("expected watcher stats")
	}
	if stats.ActiveWatches != 0 {
		t.Fatalf("expected watcher active watches to be 0 after watched stop, got %d", stats.ActiveWatches)
	}
}

func TestMetricsWatchCleanupOnWatcherStop(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))

	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))
	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
	}))

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watcher")
	<-watcher.Done()
	time.Sleep(50 * time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalWatches != 0 {
		t.Fatalf("expected 0 watches after watcher stop, got %d", snapshot.TotalWatches)
	}

	stats, ok := snapshot.ActorStats["watcher"]
	if !ok {
		t.Fatal("expected watcher stats")
	}
	if stats.ActiveWatches != 0 {
		t.Fatalf("expected watcher active watches to be 0 after watcher stop, got %d", stats.ActiveWatches)
	}
}

func TestMetricsActorStats(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))

	ref1, _ := engine.Spawn("actor-1", actorFunc(func(ctx actor.Context) {}))
	ref2, _ := engine.Spawn("actor-2", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "panic" {
			panic("test panic")
		}
	}))

	for i := 0; i < 5; i++ {
		ref1.Tell(pingMsg{})
	}
	for i := 0; i < 3; i++ {
		ref2.Tell(pingMsg{})
	}
	ref2.Tell("panic")

	time.Sleep(100 * time.Millisecond)

	snapshot := metrics.GetSnapshot()

	stats1, ok := snapshot.ActorStats["actor-1"]
	if !ok {
		t.Fatal("expected actor-1 stats")
	}
	if stats1.MessagesProcessed < 5 {
		t.Fatalf("expected at least 5 messages for actor-1, got %d", stats1.MessagesProcessed)
	}
	if stats1.Panics != 0 {
		t.Fatalf("expected 0 panics for actor-1, got %d", stats1.Panics)
	}

	stats2, ok := snapshot.ActorStats["actor-2"]
	if !ok {
		t.Fatal("expected actor-2 stats")
	}
	if stats2.MessagesProcessed < 3 {
		t.Fatalf("expected at least 3 messages for actor-2, got %d", stats2.MessagesProcessed)
	}
	if stats2.Panics != 1 {
		t.Fatalf("expected 1 panic for actor-2, got %d", stats2.Panics)
	}
}

func TestMetricsTrackBackpressureAndState(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(
		actor.WithMetrics(metrics),
		actor.WithMailboxSize(1),
		actor.WithBackpressurePolicy(actor.BackpressureDropNewest),
	)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	ref, _ := engine.Spawn("slow", actorFunc(func(ctx actor.Context) {
		once.Do(func() {
			close(started)
			<-release
		})
	}))

	if err := ref.Tell("msg-1"); err != nil {
		t.Fatalf("first tell failed: %v", err)
	}
	<-started

	if err := ref.Tell("msg-2"); err != nil {
		t.Fatalf("second tell failed: %v", err)
	}
	if err := ref.Tell("msg-3"); err == nil {
		t.Fatal("third tell should fail when mailbox is full under drop policy")
	} else {
		var fullErr *actor.MailboxFullError
		if !errors.As(err, &fullErr) {
			t.Fatalf("expected MailboxFullError, got %T", err)
		}
	}

	close(release)
	time.Sleep(100 * time.Millisecond)
	engine.Stop("slow")
	<-ref.Done()

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalDropped == 0 {
		t.Fatal("expected dropped messages to be tracked")
	}
	if len(snapshot.RecentEvents) == 0 {
		t.Fatal("expected recent metric events to be recorded")
	}

	stats, ok := snapshot.ActorStats["slow"]
	if !ok {
		t.Fatal("expected slow actor stats")
	}
	if stats.MessagesDropped == 0 {
		t.Fatal("expected per-actor dropped messages to be tracked")
	}
	if stats.MaxQueueDepth == 0 {
		t.Fatal("expected queue depth to be tracked")
	}
	if stats.State != actor.ActorStateStopped {
		t.Fatalf("expected final state stopped, got %s", stats.State)
	}
}

func TestMailboxPolicyFuncRejectsWhenFull(t *testing.T) {
	engine := actor.NewEngine(
		actor.WithMailboxSize(1),
		actor.WithMailboxPolicy(actor.MailboxPolicyFunc(func(state actor.MailboxState, incoming actor.Envelope) actor.MailboxDecision {
			return actor.MailboxDecision{
				Action: actor.MailboxActionReject,
				Reason: actor.MailboxDropReasonPolicy,
			}
		})),
	)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var processed []string
	var mu sync.Mutex

	ref, _ := engine.Spawn("policy-reject", actorFunc(func(ctx actor.Context) {
		msg, _ := ctx.Message().(string)
		mu.Lock()
		processed = append(processed, msg)
		mu.Unlock()
		once.Do(func() {
			close(started)
			<-release
		})
	}))

	if err := ref.Tell("msg-1"); err != nil {
		t.Fatalf("first tell failed: %v", err)
	}
	<-started
	if err := ref.Tell("msg-2"); err != nil {
		t.Fatalf("second tell failed: %v", err)
	}
	if err := ref.Tell("msg-3"); err == nil {
		t.Fatal("expected policy to reject the third message")
	} else {
		var rejectedErr *actor.MailboxRejectedError
		if !errors.As(err, &rejectedErr) {
			t.Fatalf("expected MailboxRejectedError, got %T", err)
		}
		if rejectedErr.Reason != actor.MailboxDropReasonPolicy {
			t.Fatalf("unexpected reject reason: %s", rejectedErr.Reason)
		}
	}

	close(release)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(processed) != 2 || processed[0] != "msg-1" || processed[1] != "msg-2" {
		t.Fatalf("expected processed messages [msg-1 msg-2], got %v", processed)
	}
}

func TestMailboxPolicyDropOldestEmitsDeadLetter(t *testing.T) {
	var (
		deadLetters []actor.DeadLetter
		deadMu      sync.Mutex
		processed   []string
		processedMu sync.Mutex
	)

	engine := actor.NewEngine(
		actor.WithMailboxSize(1),
		actor.WithMailboxPolicy(actor.MailboxPolicyFunc(func(state actor.MailboxState, incoming actor.Envelope) actor.MailboxDecision {
			return actor.MailboxDecision{Action: actor.MailboxActionDropOldest}
		})),
		actor.WithDeadLetterHandler(func(letter actor.DeadLetter) {
			deadMu.Lock()
			deadLetters = append(deadLetters, letter)
			deadMu.Unlock()
		}),
	)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	ref, _ := engine.Spawn("drop-oldest", actorFunc(func(ctx actor.Context) {
		msg, _ := ctx.Message().(string)
		processedMu.Lock()
		processed = append(processed, msg)
		processedMu.Unlock()
		once.Do(func() {
			close(started)
			<-release
		})
	}))

	if err := ref.Tell("msg-1"); err != nil {
		t.Fatalf("first tell failed: %v", err)
	}
	<-started
	if err := ref.Tell("msg-2"); err != nil {
		t.Fatalf("second tell failed: %v", err)
	}
	if err := ref.Tell("msg-3"); err != nil {
		t.Fatalf("third tell should replace oldest queued message, got: %v", err)
	}

	close(release)
	time.Sleep(100 * time.Millisecond)

	processedMu.Lock()
	gotProcessed := append([]string(nil), processed...)
	processedMu.Unlock()
	if len(gotProcessed) != 2 || gotProcessed[0] != "msg-1" || gotProcessed[1] != "msg-3" {
		t.Fatalf("expected processed messages [msg-1 msg-3], got %v", gotProcessed)
	}

	deadMu.Lock()
	defer deadMu.Unlock()
	if len(deadLetters) != 1 {
		t.Fatalf("expected 1 dead letter, got %d", len(deadLetters))
	}
	msg, _ := deadLetters[0].Envelope.Message.(string)
	if msg != "msg-2" {
		t.Fatalf("expected dead letter message msg-2, got %v", deadLetters[0].Envelope.Message)
	}
	if deadLetters[0].Reason != actor.MailboxDropReasonEvicted {
		t.Fatalf("expected dead letter reason evicted, got %s", deadLetters[0].Reason)
	}
	if deadLetters[0].Target != "drop-oldest" {
		t.Fatalf("expected dead letter target drop-oldest, got %s", deadLetters[0].Target)
	}
}

func TestEngineMetricsMethod(t *testing.T) {
	metrics := actor.NewDefaultMetrics()
	engine := actor.NewEngine(actor.WithMetrics(metrics))

	if engine.Metrics() != metrics {
		t.Fatal("engine.Metrics() should return the same metrics instance")
	}

	defaultEngine := actor.NewEngine()
	if defaultEngine.Metrics() == nil {
		t.Fatal("default engine should have a noop metrics instance")
	}
}

func TestUnwatchNotWatching(t *testing.T) {
	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "unwatch" {
			ctx.Unwatch(watched)
		}
	}))

	time.Sleep(50 * time.Millisecond)

	watcher.Tell("unwatch")
	time.Sleep(50 * time.Millisecond)
}

func TestUnwatchNonExistentWatched(t *testing.T) {
	engine := actor.NewEngine()
	dummy, _ := engine.Spawn("dummy", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "unwatch" {
			ctx.Unwatch(dummy)
		}
	}))

	time.Sleep(50 * time.Millisecond)

	engine.Stop("dummy")
	<-dummy.Done()

	watcher.Tell("unwatch")
	time.Sleep(50 * time.Millisecond)
}

func TestWatcherStopCleansUpWatch(t *testing.T) {
	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
	}))

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watcher")
	time.Sleep(100 * time.Millisecond)

	var terminatedCount atomic.Int32
	watcher2, _ := engine.Spawn("watcher2", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminatedCount.Add(1)
		}
	}))

	watcher2.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if terminatedCount.Load() != 1 {
		t.Fatalf("expected watcher2 to receive 1 Terminated, got %d", terminatedCount.Load())
	}
}

func TestWatcherStopWithMultipleWatched(t *testing.T) {
	engine := actor.NewEngine()
	watched1, _ := engine.Spawn("watched1", actorFunc(func(ctx actor.Context) {}))
	watched2, _ := engine.Spawn("watched2", actorFunc(func(ctx actor.Context) {}))

	var terminatedCount atomic.Int32
	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched1)
			ctx.Watch(watched2)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminatedCount.Add(1)
		}
	}))

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watcher")
	time.Sleep(100 * time.Millisecond)

	engine.Stop("watched1")
	engine.Stop("watched2")
	time.Sleep(100 * time.Millisecond)

	if terminatedCount.Load() != 0 {
		t.Fatalf("stopped watcher should not receive Terminated, got %d", terminatedCount.Load())
	}
}

func TestMultipleWatchersOneUnwatches(t *testing.T) {
	var terminated1, terminated2 atomic.Bool

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher1, _ := engine.Spawn("watcher1", actorFunc(func(ctx actor.Context) {
		switch ctx.Message() {
		case "watch":
			ctx.Watch(watched)
		case "unwatch":
			ctx.Unwatch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated1.Store(true)
		}
	}))

	watcher2, _ := engine.Spawn("watcher2", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated2.Store(true)
		}
	}))

	watcher1.Tell("watch")
	watcher2.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	watcher1.Tell("unwatch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if terminated1.Load() {
		t.Fatal("watcher1 should not receive Terminated after unwatch")
	}
	if !terminated2.Load() {
		t.Fatal("watcher2 should receive Terminated")
	}
}

func TestMultipleWatchersAllStop(t *testing.T) {
	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher1, _ := engine.Spawn("watcher1", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
	}))
	watcher2, _ := engine.Spawn("watcher2", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
	}))

	watcher1.Tell("watch")
	watcher2.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watcher1")
	engine.Stop("watcher2")
	time.Sleep(100 * time.Millisecond)

	var terminated atomic.Bool
	watcher3, _ := engine.Spawn("watcher3", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated.Store(true)
		}
	}))

	watcher3.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if !terminated.Load() {
		t.Fatal("watcher3 should receive Terminated after all previous watchers stopped")
	}
}

func TestWatchConcurrentWatchUnwatchStop(t *testing.T) {
	const iterations = 50
	engine := actor.NewEngine()

	for i := 0; i < iterations; i++ {
		watched, _ := engine.Spawn(fmt.Sprintf("watched-%d", i), actorFunc(func(ctx actor.Context) {}))

		var terminatedCount atomic.Int32
		var wg sync.WaitGroup

		for j := 0; j < 3; j++ {
			wg.Add(1)
			watcher, _ := engine.Spawn(fmt.Sprintf("watcher-%d-%d", i, j), actorFunc(func(ctx actor.Context) {
				switch ctx.Message() {
				case "watch":
					ctx.Watch(watched)
				case "unwatch":
					ctx.Unwatch(watched)
				}
				if _, ok := ctx.Message().(actor.Terminated); ok {
					terminatedCount.Add(1)
				}
			}))
			watcher.Tell("watch")
			time.Sleep(5 * time.Millisecond)

			if j == 0 {
				go func() {
					defer wg.Done()
					watcher.Tell("unwatch")
				}()
			} else {
				wg.Done()
			}
		}

		go func() {
			time.Sleep(10 * time.Millisecond)
			engine.Stop(watched.Name())
		}()

		wg.Wait()
		time.Sleep(30 * time.Millisecond)

		count := terminatedCount.Load()
		if count > 2 {
			t.Fatalf("iteration %d: expected at most 2 Terminated, got %d", i, count)
		}
	}
}

func TestSpawnAfterShutdown(t *testing.T) {
	engine := actor.NewEngine()
	_, _ = engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))

	err := engine.Shutdown()
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	_, err = engine.Spawn("new-actor", actorFunc(func(ctx actor.Context) {}))
	if err != actor.ErrEngineShutdown {
		t.Fatalf("expected ErrEngineShutdown, got: %v", err)
	}
}

func TestStrategyStop(t *testing.T) {
	var stopped atomic.Bool

	engine := actor.NewEngine(actor.WithSupervisionStrategy(actor.StrategyStop))
	ref, _ := engine.Spawn("panicker", &lifecycleActor{
		actorFunc: actorFunc(func(ctx actor.Context) {
			if ctx.Message() == "boom" {
				panic("test panic")
			}
		}),
		lifecycleFuncs: lifecycleFuncs{
			onStart: func(ctx actor.Context) {},
			onStop:  func(ctx actor.Context) { stopped.Store(true) },
		},
	})

	ref.Tell("boom")
	<-ref.Done()

	if !stopped.Load() {
		t.Fatal("actor should be stopped after panic with StrategyStop")
	}

	if err := ref.Tell(pingMsg{}); err == nil {
		t.Fatal("expected error when telling stopped actor")
	}
}

func TestStrategyStopRemovesActorFromRegistry(t *testing.T) {
	engine := actor.NewEngine(actor.WithSupervisionStrategy(actor.StrategyStop))
	ref, _ := engine.Spawn("panicker", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "boom" {
			panic("test panic")
		}
	}))

	ref.Tell("boom")
	<-ref.Done()

	if _, ok := engine.Lookup("panicker"); ok {
		t.Fatal("strategy stop should remove actor from registry")
	}

	respawned, err := engine.Spawn("panicker", actorFunc(func(ctx actor.Context) {}))
	if err != nil {
		t.Fatalf("expected respawn after strategy stop to succeed: %v", err)
	}
	engine.Stop(respawned.Name())
	<-respawned.Done()
}

func TestStrategyResume(t *testing.T) {
	var received atomic.Int32

	engine := actor.NewEngine(actor.WithSupervisionStrategy(actor.StrategyResume))
	ref, _ := engine.Spawn("panicker", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "boom" {
			panic("test panic")
		}
		if ctx.Message() == "safe" {
			received.Add(1)
		}
	}))

	ref.Tell("boom")
	time.Sleep(50 * time.Millisecond)

	ref.Tell("safe")
	time.Sleep(50 * time.Millisecond)

	if received.Load() != 1 {
		t.Fatalf("expected 1 safe message after resume, got %d", received.Load())
	}
}

func TestStrategyRestart(t *testing.T) {
	var started, stopped, safe atomic.Int32

	engine := actor.NewEngine(
		actor.WithSupervisionStrategy(actor.StrategyRestart),
		actor.WithRestartPolicy(2, 10*time.Millisecond, time.Second),
		actor.WithMetrics(actor.NewDefaultMetrics()),
	)
	ref, _ := engine.Spawn("restartable", &restartablePanicActor{
		started: &started,
		stopped: &stopped,
		safe:    &safe,
	})

	ref.Tell("boom")
	time.Sleep(80 * time.Millisecond)
	ref.Tell("safe")
	time.Sleep(80 * time.Millisecond)

	if started.Load() < 2 {
		t.Fatalf("expected actor to start at least twice, got %d", started.Load())
	}
	if stopped.Load() < 1 {
		t.Fatalf("expected actor stop hook during restart, got %d", stopped.Load())
	}
	if safe.Load() != 1 {
		t.Fatalf("expected safe message after restart, got %d", safe.Load())
	}

	snapshot := engine.Metrics().GetSnapshot()
	if snapshot.TotalRestarts != 1 {
		t.Fatalf("expected 1 restart, got %d", snapshot.TotalRestarts)
	}

	stats, ok := snapshot.ActorStats["restartable"]
	if !ok {
		t.Fatal("expected restartable actor stats")
	}
	if stats.Restarts != 1 {
		t.Fatalf("expected per-actor restarts to be 1, got %d", stats.Restarts)
	}
	if stats.State != actor.ActorStateRunning {
		t.Fatalf("expected actor to be running after restart, got %s", stats.State)
	}
}

func TestPoisonPillTellFails(t *testing.T) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("test", actorFunc(func(ctx actor.Context) {}))

	ref.Tell(actor.PoisonPill{})
	<-ref.Done()

	if err := ref.Tell(pingMsg{}); err == nil {
		t.Fatal("Tell after PoisonPill should return error")
	}
}
