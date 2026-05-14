package actor_test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"collab-actor-demo/internal/actor"
)

func TestStabilityManyActors(t *testing.T) {
	engine := actor.NewEngine()
	const count = 1000

	var started, stopped atomic.Int32

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("actor-%d", i)
		_, err := engine.Spawn(name, &lifecycleActor{
			actorFunc: actorFunc(func(ctx actor.Context) {}),
			lifecycleFuncs: lifecycleFuncs{
				onStart: func(ctx actor.Context) { started.Add(1) },
				onStop:  func(ctx actor.Context) { stopped.Add(1) },
			},
		})
		if err != nil {
			t.Fatalf("spawn %s failed: %v", name, err)
		}
	}

	time.Sleep(500 * time.Millisecond)
	if started.Load() != count {
		t.Fatalf("expected %d started, got %d", count, started.Load())
	}

	for i := 0; i < count; i++ {
		engine.Stop(fmt.Sprintf("actor-%d", i))
	}

	time.Sleep(500 * time.Millisecond)
	if stopped.Load() != count {
		t.Fatalf("expected %d stopped, got %d", count, stopped.Load())
	}
}

func TestStabilityRepeatedPanic(t *testing.T) {
	engine := actor.NewEngine()
	var received atomic.Int32

	ref, _ := engine.Spawn("panicker", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "boom" {
			panic("repeated panic")
		}
		if ctx.Message() == "safe" {
			received.Add(1)
		}
	}))

	for i := 0; i < 100; i++ {
		ref.Tell("boom")
		ref.Tell("safe")
	}

	time.Sleep(500 * time.Millisecond)

	if received.Load() != 100 {
		t.Fatalf("expected 100 safe messages, got %d", received.Load())
	}
}

func TestStabilityGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	engine := actor.NewEngine()
	for i := 0; i < 100; i++ {
		ref, _ := engine.Spawn(fmt.Sprintf("actor-%d", i), actorFunc(func(ctx actor.Context) {}))
		engine.Stop(ref.Name())
		<-ref.Done()
	}

	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - before

	if leaked > 10 {
		t.Fatalf("potential goroutine leak: %d goroutines leaked (before=%d, after=%d)", leaked, before, after)
	}
}

func TestStabilityWatchGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))
	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
		if ctx.Message() == "unwatch" {
			ctx.Unwatch(watched)
		}
	}))

	for i := 0; i < 50; i++ {
		watcher.Tell("watch")
		time.Sleep(10 * time.Millisecond)
		watcher.Tell("unwatch")
		time.Sleep(10 * time.Millisecond)
	}

	engine.Stop("watched")
	engine.Stop("watcher")
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - before

	if leaked > 10 {
		t.Fatalf("potential watch goroutine leak: %d goroutines leaked (before=%d, after=%d)", leaked, before, after)
	}
}

func TestStabilityHighFrequencyTell(t *testing.T) {
	engine := actor.NewEngine()
	var received atomic.Int64

	ref, _ := engine.Spawn("sink", actorFunc(func(ctx actor.Context) {
		received.Add(1)
	}))

	const total = 100000
	for i := 0; i < total; i++ {
		ref.Tell("msg")
	}

	deadline := time.Now().Add(5 * time.Second)
	for received.Load() < total && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if received.Load() != total {
		t.Fatalf("expected %d messages, got %d", total, received.Load())
	}
}

func TestStabilityConcurrentSpawnStop(t *testing.T) {
	engine := actor.NewEngine()
	const workers = 10
	const opsPerWorker = 100

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				name := fmt.Sprintf("w%d-%d", workerID, i)
				ref, err := engine.Spawn(name, actorFunc(func(ctx actor.Context) {}))
				if err != nil {
					continue
				}
				engine.Stop(name)
				<-ref.Done()
			}
		}(w)
	}

	wg.Wait()
}
