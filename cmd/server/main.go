package main

import (
	"fmt"
	"time"

	"collab-actor-demo/internal/actor"
)

type EditMsg struct {
	Text string
}

type GetContentMsg struct{}

type ContentResult struct {
	Content string
}

type DocActor struct {
	content string
	version int
}

func (d *DocActor) OnStart(ctx actor.Context) {
	d.content = ""
	d.version = 0
	fmt.Printf("[DocActor] %s started\n", ctx.Self().Name())
}

func (d *DocActor) OnStop(ctx actor.Context) {
	fmt.Printf("[DocActor] %s stopped, final content: %q\n", ctx.Self().Name(), d.content)
}

func (d *DocActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case EditMsg:
		d.content += msg.Text
		d.version++
		fmt.Printf("[DocActor] %s received edit: %q, version=%d, content=%q\n",
			ctx.Self().Name(), msg.Text, d.version, d.content)

		if ctx.Sender() != nil {
			ctx.Sender().Tell(fmt.Sprintf("ack v%d", d.version))
		}

	case GetContentMsg:
		ctx.Reply(ContentResult{Content: d.content})
	}
}

type UserActor struct {
	name string
}

func (u *UserActor) OnStart(ctx actor.Context) {
	fmt.Printf("[UserActor] %s online\n", u.name)
}

func (u *UserActor) OnStop(ctx actor.Context) {
	fmt.Printf("[UserActor] %s offline\n", u.name)
}

func (u *UserActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case string:
		fmt.Printf("[UserActor] %s received: %s\n", u.name, msg)
	}
}

func main() {
	engine := actor.NewEngine()

	docRef, err := engine.Spawn("doc-1", &DocActor{})
	if err != nil {
		fmt.Println("spawn doc failed:", err)
		return
	}

	userRef, err := engine.Spawn("user-alice", &UserActor{name: "alice"})
	if err != nil {
		fmt.Println("spawn user failed:", err)
		return
	}

	fmt.Println("\n--- Tell: alice 编辑文档 ---")
	docRef.TellFrom(userRef, EditMsg{Text: "Hello "})

	fmt.Println("\n--- Tell: 匿名编辑 ---")
	docRef.Tell(EditMsg{Text: "World"})

	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n--- Ask: 查询文档内容 ---")
	result, err := docRef.Ask(GetContentMsg{}, 3*time.Second)
	if err != nil {
		fmt.Println("ask failed:", err)
		return
	}
	if cr, ok := result.(ContentResult); ok {
		fmt.Printf("[Main] document content: %q\n", cr.Content)
	}

	fmt.Println("\n--- Lookup: 通过名字查找 ---")
	if ref, ok := engine.Lookup("doc-1"); ok {
		ref.Tell(EditMsg{Text: "!"})
	}

	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n--- Spawn 重复名 ---")
	_, err = engine.Spawn("doc-1", &DocActor{})
	fmt.Println("expected error:", err)

	fmt.Println("\n--- Stop: 停止 Actor ---")
	engine.Stop("doc-1")
	engine.Stop("user-alice")

	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n--- 向已停止的 Actor 发消息 ---")
	err = docRef.Tell(EditMsg{Text: "should fail"})
	fmt.Println("expected error:", err)

	fmt.Println("\n--- Reply on non-Ask message ---")
	badRef, _ := engine.Spawn("bad-actor", &DocActor{})
	time.Sleep(50 * time.Millisecond)
	badRef.Tell(GetContentMsg{})
	time.Sleep(100 * time.Millisecond)
}
