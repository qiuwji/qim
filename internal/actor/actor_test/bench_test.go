package actor_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"qim/internal/actor"
)

type benchMsg struct{}

type benchActor struct {
	received atomic.Int64
}

func (a *benchActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case benchMsg:
		a.received.Add(1)
	}
}

type echoActor struct{}

func (e *echoActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case benchMsg:
		ctx.Reply(benchMsg{})
	}
}

func waitForCount(b *testing.B, counter *atomic.Int64, want int64, label string) {
	b.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for counter.Load() < want {
		if time.Now().After(deadline) {
			b.Fatalf("timed out waiting for %s: got %d, want %d", label, counter.Load(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func BenchmarkTellThroughput(b *testing.B) {
	engine := actor.NewEngine()
	a := &benchActor{}
	ref, _ := engine.Spawn("bench", a)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ref.Tell(benchMsg{})
	}

	waitForCount(b, &a.received, int64(b.N), "tell throughput messages")
	b.StopTimer()
}

func BenchmarkTellParallel(b *testing.B) {
	engine := actor.NewEngine()
	a := &benchActor{}
	ref, _ := engine.Spawn("bench", a)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ref.Tell(benchMsg{})
		}
	})

	waitForCount(b, &a.received, int64(b.N), "parallel tell messages")
	b.StopTimer()
}

func BenchmarkAskThroughput(b *testing.B) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("bench", &echoActor{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ref.Ask(benchMsg{}, 5*time.Second)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAskParallel(b *testing.B) {
	engine := actor.NewEngine()
	ref, _ := engine.Spawn("bench", &echoActor{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ref.Ask(benchMsg{}, 5*time.Second)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSpawnStop(b *testing.B) {
	engine := actor.NewEngine()
	a := &benchActor{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("actor-%d", i)
		ref, err := engine.Spawn(name, a)
		if err != nil {
			b.Fatal(err)
		}
		engine.Stop(name)
		<-ref.Done()
	}
}

func BenchmarkLookup(b *testing.B) {
	engine := actor.NewEngine()
	_, _ = engine.Spawn("bench", &benchActor{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, ok := engine.Lookup("bench")
		if !ok {
			b.Fatal("actor not found")
		}
	}
}

func BenchmarkLookupParallel(b *testing.B) {
	engine := actor.NewEngine()
	_, _ = engine.Spawn("bench", &benchActor{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, ok := engine.Lookup("bench")
			if !ok {
				b.Fatal("actor not found")
			}
		}
	})
}

func BenchmarkWatch(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := actor.NewEngine()
		watched, _ := engine.Spawn(fmt.Sprintf("watched-%d", i), &benchActor{})
		watcher, _ := engine.Spawn(fmt.Sprintf("watcher-%d", i), actorFunc(func(ctx actor.Context) {
			if ctx.Message() == "start" {
				ctx.Watch(watched)
			}
		}))
		watcher.Tell("start")
		time.Sleep(time.Millisecond)
		engine.Stop(watched.Name())
		<-watched.Done()
	}
}

func BenchmarkMultiActorTell(b *testing.B) {
	engine := actor.NewEngine()
	actors := make([]*actor.ActorRef, 100)
	for i := 0; i < 100; i++ {
		ref, _ := engine.Spawn(fmt.Sprintf("actor-%d", i), &benchActor{})
		actors[i] = ref
	}

	var total atomic.Int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		actors[i%100].Tell(benchMsg{})
		total.Add(1)
	}
	b.StopTimer()
}

func BenchmarkPipeline(b *testing.B) {
	engine := actor.NewEngine()

	var endReceived atomic.Int64

	stage3, _ := engine.Spawn("stage3", actorFunc(func(ctx actor.Context) {
		endReceived.Add(1)
	}))

	stage2, _ := engine.Spawn("stage2", actorFunc(func(ctx actor.Context) {
		stage3.Tell(benchMsg{})
	}))

	stage1, _ := engine.Spawn("stage1", actorFunc(func(ctx actor.Context) {
		stage2.Tell(benchMsg{})
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stage1.Tell(benchMsg{})
	}

	waitForCount(b, &endReceived, int64(b.N), "pipeline messages")
	b.StopTimer()
}

func BenchmarkFanOut(b *testing.B) {
	engine := actor.NewEngine()
	var received atomic.Int64

	actors := make([]*actor.ActorRef, 10)
	for i := 0; i < 10; i++ {
		ref, _ := engine.Spawn(fmt.Sprintf("fan-%d", i), actorFunc(func(ctx actor.Context) {
			received.Add(1)
		}))
		actors[i] = ref
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ref := range actors {
			ref.Tell(benchMsg{})
		}
	}

	waitForCount(b, &received, int64(b.N)*10, "fan-out messages")
	b.StopTimer()
}

func BenchmarkMiddlewareOverhead(b *testing.B) {
	noop := func(next actor.ReceiveFunc) actor.ReceiveFunc {
		return func(ctx actor.Context) {
			next(ctx)
		}
	}

	engine := actor.NewEngine(actor.WithMiddleware(noop, noop, noop))
	a := &benchActor{}
	ref, _ := engine.Spawn("bench", a)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ref.Tell(benchMsg{})
	}

	waitForCount(b, &a.received, int64(b.N), "middleware messages")
	b.StopTimer()
}

func BenchmarkScheduleAfter(b *testing.B) {
	engine := actor.NewEngine()
	var received atomic.Int64

	ref, _ := engine.Spawn("bench", actorFunc(func(ctx actor.Context) {
		switch ctx.Message() {
		case "schedule":
			ctx.ScheduleAfter(time.Microsecond, "tick")
		case "tick":
			received.Add(1)
		}
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ref.Tell("schedule")
	}

	waitForCount(b, &received, int64(b.N), "scheduled messages")
	b.StopTimer()
}
