package actor_test

import (
	"sync/atomic"
	"testing"
	"time"

	"collab-actor-demo/internal/actor"
)

func TestMonitorNilStateRace(t *testing.T) {
	const iterations = 100
	for i := 0; i < iterations; i++ {
		engine := actor.NewEngine()

		watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

		var terminated atomic.Int32
		watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
			if ctx.Message() == "watch" {
				ctx.Watch(watched)
			}
			if ctx.Message() == "unwatch" {
				ctx.Unwatch(watched)
			}
			if _, ok := ctx.Message().(actor.Terminated); ok {
				terminated.Add(1)
			}
		}))

		watcher.Tell("watch")
		time.Sleep(2 * time.Millisecond)

		go watcher.Tell("unwatch")
		go engine.Stop("watched")

		time.Sleep(20 * time.Millisecond)

		if terminated.Load() > 1 {
			t.Fatalf("iteration %d: expected at most 1 Terminated, got %d", i, terminated.Load())
		}
	}
}

func TestMonitorStopChEarlyExit(t *testing.T) {
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

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	watcher.Tell("unwatch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	<-watched.Done()
}

func TestMonitorNormalTermination(t *testing.T) {
	var terminated atomic.Bool

	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	watcher, _ := engine.Spawn("watcher", actorFunc(func(ctx actor.Context) {
		if ctx.Message() == "watch" {
			ctx.Watch(watched)
		}
		if _, ok := ctx.Message().(actor.Terminated); ok {
			terminated.Store(true)
		}
	}))

	watcher.Tell("watch")
	time.Sleep(50 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if !terminated.Load() {
		t.Fatal("watcher should receive Terminated")
	}

	if _, exists := engine.Lookup("watched"); exists {
		t.Fatal("watched should be removed from engine after termination")
	}
}

func TestUnwatchThenRewatch(t *testing.T) {
	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	var terminated1, terminated2 atomic.Bool

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
	time.Sleep(20 * time.Millisecond)

	watcher1.Tell("unwatch")
	time.Sleep(20 * time.Millisecond)

	watcher2.Tell("watch")
	time.Sleep(20 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if terminated1.Load() {
		t.Fatal("watcher1 should not receive Terminated after unwatch")
	}
	if !terminated2.Load() {
		t.Fatal("watcher2 should receive Terminated")
	}
}

func TestAllUnwatchThenNewWatch(t *testing.T) {
	engine := actor.NewEngine()
	watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

	var terminated1, terminated2 atomic.Bool

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
	time.Sleep(20 * time.Millisecond)

	watcher1.Tell("unwatch")
	time.Sleep(20 * time.Millisecond)

	watcher2.Tell("watch")
	time.Sleep(20 * time.Millisecond)

	engine.Stop("watched")
	time.Sleep(100 * time.Millisecond)

	if terminated1.Load() {
		t.Fatal("watcher1 should not receive Terminated after unwatch")
	}
	if !terminated2.Load() {
		t.Fatal("watcher2 should receive Terminated — old monitor must not corrupt new state")
	}
}

func TestConcurrentUnwatchRewatchStop(t *testing.T) {
	const iterations = 200
	for i := 0; i < iterations; i++ {
		engine := actor.NewEngine()
		watched, _ := engine.Spawn("watched", actorFunc(func(ctx actor.Context) {}))

		var watcher1Terminated, watcher2Terminated atomic.Int32

		watcher1, _ := engine.Spawn("watcher1", actorFunc(func(ctx actor.Context) {
			switch ctx.Message() {
			case "watch":
				ctx.Watch(watched)
			case "unwatch":
				ctx.Unwatch(watched)
			}
			if _, ok := ctx.Message().(actor.Terminated); ok {
				watcher1Terminated.Add(1)
			}
		}))

		watcher2, _ := engine.Spawn("watcher2", actorFunc(func(ctx actor.Context) {
			if ctx.Message() == "watch" {
				ctx.Watch(watched)
			}
			if _, ok := ctx.Message().(actor.Terminated); ok {
				watcher2Terminated.Add(1)
			}
		}))

		watcher1.Tell("watch")
		time.Sleep(time.Millisecond)

		go watcher1.Tell("unwatch")
		go watcher2.Tell("watch")
		go engine.Stop("watched")

		time.Sleep(30 * time.Millisecond)

		if watcher1Terminated.Load() > 1 {
			t.Fatalf("iteration %d: watcher1 received %d Terminated, expected at most 1", i, watcher1Terminated.Load())
		}
		if watcher2Terminated.Load() > 1 {
			t.Fatalf("iteration %d: watcher2 received %d Terminated, expected at most 1", i, watcher2Terminated.Load())
		}
	}
}
