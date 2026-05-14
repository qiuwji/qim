package main

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"collab-actor-demo/internal/actor"
)

type JoinMsg struct {
	User string
}

type LeaveMsg struct {
	User string
}

type ChatMsg struct {
	From string
	Text string
}

type HeartbeatMsg struct{}

type UserOnline struct {
	Name string
}

type UserOffline struct {
	Name string
	Reason string
}

type RoomActor struct {
	name    string
	members map[string]*actor.ActorRef
	history []string
}

func newRoomActor(name string) *RoomActor {
	return &RoomActor{name: name, members: make(map[string]*actor.ActorRef)}
}

func (r *RoomActor) OnStart(ctx actor.Context) {
	fmt.Printf("🏠 [%s] 聊天室已创建\n", r.name)
}

func (r *RoomActor) OnStop(ctx actor.Context) {
	fmt.Printf("🏠 [%s] 聊天室已关闭，共 %d 条消息\n", r.name, len(r.history))
}

func (r *RoomActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case JoinMsg:
		r.members[msg.User] = ctx.Sender()
		ctx.Watch(ctx.Sender())
		r.broadcast(fmt.Sprintf("🟢 %s 加入了聊天室 (当前 %d 人)", msg.User, len(r.members)))
		for _, h := range r.history {
			if sender := ctx.Sender(); sender != nil {
				sender.Tell(ChatMsg{From: "系统", Text: h})
			}
		}

	case LeaveMsg:
		delete(r.members, msg.User)
		r.broadcast(fmt.Sprintf("🔴 %s 离开了聊天室 (当前 %d 人)", msg.User, len(r.members)))

	case ChatMsg:
		line := fmt.Sprintf("%s: %s", msg.From, msg.Text)
		r.history = append(r.history, line)
		r.broadcast(line)

	case actor.Terminated:
		name := msg.Who.Name()
		if _, ok := r.members[name]; ok {
			delete(r.members, name)
			r.broadcast(fmt.Sprintf("💥 %s 异常断开 (当前 %d 人)", name, len(r.members)))
		}
	}
}

func (r *RoomActor) broadcast(text string) {
	for _, ref := range r.members {
		ref.Tell(ChatMsg{From: "系统", Text: text})
	}
}

type ConnActor struct {
	user   string
	online bool
}

func newConnActor(user string) *ConnActor {
	return &ConnActor{user: user}
}

func (c *ConnActor) OnStart(ctx actor.Context) {
	c.online = true
	fmt.Printf("👤 [%s] 已连接\n", c.user)
	ctx.ScheduleAfter(2*time.Second, HeartbeatMsg{})
}

func (c *ConnActor) OnStop(ctx actor.Context) {
	c.online = false
	fmt.Printf("👤 [%s] 已断开\n", c.user)
}

func (c *ConnActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ChatMsg:
		if !c.online {
			return
		}
		if msg.From == "系统" {
			fmt.Printf("   📨 [%s] %s\n", c.user, msg.Text)
		} else {
			fmt.Printf("   💬 [%s] %s\n", c.user, msg.Text)
		}

	case HeartbeatMsg:
		if c.online {
			ctx.ScheduleAfter(2*time.Second, HeartbeatMsg{})
		}

	case UserOffline:
		fmt.Printf("   ⚠️  [%s] 收到离线通知: %s\n", c.user, msg.Reason)
		c.online = false
	}
}

type CrashMsg struct{}

type UnstableActor struct {
	name string
}

func (u *UnstableActor) OnStart(ctx actor.Context) {
	fmt.Printf("💥 [%s] 不稳定 Actor 上线\n", u.name)
}

func (u *UnstableActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case CrashMsg:
		fmt.Printf("💥 [%s] 即将崩溃...\n", u.name)
		panic("模拟崩溃!")
	case ChatMsg:
		fmt.Printf("💥 [%s] 收到消息，继续工作\n", u.name)
	}
}

var msgCount atomic.Int64

func metricsMiddleware(next actor.ReceiveFunc) actor.ReceiveFunc {
	return func(ctx actor.Context) {
		msgCount.Add(1)
		next(ctx)
	}
}

func loggingMiddleware(next actor.ReceiveFunc) actor.ReceiveFunc {
	return func(ctx actor.Context) {
		if _, ok := ctx.Message().(HeartbeatMsg); !ok {
			fmt.Printf("   🔍 [%s] 处理消息: %T\n", ctx.Self().Name(), ctx.Message())
		}
		next(ctx)
	}
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  Actor 引擎 IM 演示")
	fmt.Println(strings.Repeat("=", 60))

	engine := actor.NewEngine(
		actor.WithMiddleware(metricsMiddleware, loggingMiddleware),
		actor.WithMailboxSize(512),
	)

	fmt.Println("\n📌 场景1: 聊天室 — 消息隔离 + Watch 退出感知")
	fmt.Println(strings.Repeat("-", 60))

	roomRef, _ := engine.Spawn("golang-room", newRoomActor("Golang交流群"))

	alice, _ := engine.Spawn("alice", newConnActor("Alice"))
	bob, _ := engine.Spawn("bob", newConnActor("Bob"))
	carol, _ := engine.Spawn("carol", newConnActor("Carol"))

	time.Sleep(100 * time.Millisecond)

	roomRef.TellFrom(alice, JoinMsg{User: "alice"})
	roomRef.TellFrom(bob, JoinMsg{User: "bob"})
	roomRef.TellFrom(carol, JoinMsg{User: "carol"})
	time.Sleep(200 * time.Millisecond)

	roomRef.TellFrom(alice, ChatMsg{From: "Alice", Text: "大家好！"})
	time.Sleep(100 * time.Millisecond)
	roomRef.TellFrom(bob, ChatMsg{From: "Bob", Text: "Hi Alice!"})
	time.Sleep(100 * time.Millisecond)
	roomRef.TellFrom(carol, ChatMsg{From: "Carol", Text: "有人在写 Go 吗？"})
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n📌 场景2: Watch — 感知连接断开，自动清理")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("   ⚡ 强制断开 Bob 的连接...")
	engine.Stop("bob")
	time.Sleep(300 * time.Millisecond)

	fmt.Println("\n📌 场景3: Panic 隔离 — 一个 Actor 崩溃不影响其他人")
	fmt.Println(strings.Repeat("-", 60))

	crashy, _ := engine.Spawn("unstable", &UnstableActor{name: "Unstable"})
	time.Sleep(100 * time.Millisecond)

	crashy.Tell(CrashMsg{})
	time.Sleep(100 * time.Millisecond)

	roomRef.TellFrom(alice, ChatMsg{From: "Alice", Text: "刚才有人崩溃了？我这边一切正常"})
	time.Sleep(200 * time.Millisecond)

	crashy.Tell(ChatMsg{})
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n📌 场景4: Ask — 请求-应答模式")
	fmt.Println(strings.Repeat("-", 60))

	roomRef.TellFrom(alice, ChatMsg{From: "Alice", Text: "有人在线吗？"})
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n📌 场景5: ScheduleAfter — 心跳保活")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("   ⏳ 等待心跳周期 (2秒)...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n📌 场景6: Middleware — 消息统计")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("   📊 总消息处理量: %d 条\n", msgCount.Load())

	fmt.Println("\n📌 场景7: 优雅关闭 — Shutdown 等待所有 Actor 退出")
	fmt.Println(strings.Repeat("-", 60))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := engine.ShutdownWithContext(ctx)
	if err != nil {
		fmt.Printf("   ❌ 关闭超时: %v\n", err)
	} else {
		fmt.Println("   ✅ 所有 Actor 已优雅退出")
	}

	fmt.Printf("   📊 最终消息处理量: %d 条\n", msgCount.Load())

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  演示完成")
	fmt.Println(strings.Repeat("=", 60))
}
