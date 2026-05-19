package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/domain/conversation"
	"qim/internal/eventbus"
	"qim/internal/pkg/logx"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db := initDB()
	convStore := dal.NewConvStore(db)
	engine := initEngine()
	events := initEventBus(engine)

	mustSpawn(engine, "conv-manager", conversation.NewManagerActor(convStore, engine, events))

	uid1 := uint64(900001)
	uid2 := uint64(900002)

	fmt.Println("==> 创建单聊会话")
	managerRef, _ := engine.Lookup("conv-manager")
	raw, err := managerRef.Ask(conversation.CreatePrivateConvCmd{UID1: uid1, UID2: uid2}, 5*time.Second)
	if err != nil {
		fmt.Printf("创建会话失败: %v\n", err)
		return
	}
	convResult, ok := raw.(conversation.Result)
	if !ok || convResult.Err != nil {
		fmt.Printf("创建会话返回异常: %v %v\n", ok, convResult.Err)
		return
	}
	convDTO := convResult.Data.(conversation.ConversationDTO)
	convID := convDTO.ID
	fmt.Printf("    会话 ID: %d\n", convID)

	convRef, _ := engine.Lookup(fmt.Sprintf("conv:%d", convID))

	fmt.Println("==> 预热: 发送 5 条消息")
	for i := 0; i < 5; i++ {
		_, err := convRef.Ask(conversation.SendMessageCmd{
			SenderID: uid1,
			MsgType:  1,
			Content:  fmt.Sprintf("warmup-%d", i),
			ClientID: fmt.Sprintf("warmup-%d", i),
		}, 5*time.Second)
		if err != nil {
			fmt.Printf("预热消息 %d 失败: %v\n", i, err)
		}
	}

	totalMsgs := 1000
	concurrency := 10
	msgsPerWorker := totalMsgs / concurrency

	fmt.Printf("==> 开始压测: %d 并发 x %d 消息/协程 = %d 消息\n", concurrency, msgsPerWorker, totalMsgs)

	var successCount int64
	var failCount int64
	var totalLatencyNs int64

	start := time.Now()

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			sender := uid1
			if workerID%2 == 1 {
				sender = uid2
			}
			for i := 0; i < msgsPerWorker; i++ {
				msgStart := time.Now()
				_, err := convRef.Ask(conversation.SendMessageCmd{
					SenderID: sender,
					MsgType:  1,
					Content:  fmt.Sprintf("bench-w%d-i%d", workerID, i),
					ClientID: fmt.Sprintf("bench-w%d-i%d-%d", workerID, i, msgStart.UnixNano()),
				}, 5*time.Second)
				latency := time.Since(msgStart)
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					continue
				}
				atomic.AddInt64(&totalLatencyNs, latency.Nanoseconds())
				atomic.AddInt64(&successCount, 1)
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)

	succ := atomic.LoadInt64(&successCount)
	fail := atomic.LoadInt64(&failCount)
	avgLatencyMs := float64(0)
	if succ > 0 {
		avgLatencyMs = float64(atomic.LoadInt64(&totalLatencyNs)) / float64(succ) / 1e6
	}

	fmt.Println()
	fmt.Println("========== QIM 发消息写 QPS 压测结果 ==========")
	fmt.Printf("测试模式:      本地直接 Ask ConversationActor（跳过 WS/HTTP）\n")
	fmt.Printf("并发协程数:    %d\n", concurrency)
	fmt.Printf("总发送消息数:  %d\n", totalMsgs)
	fmt.Printf("成功:          %d\n", succ)
	fmt.Printf("失败:          %d\n", fail)
	fmt.Printf("总耗时:        %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("写 QPS:        %.1f msg/s\n", float64(succ)/elapsed.Seconds())
	fmt.Printf("平均延迟:      %.2f ms\n", avgLatencyMs)
	fmt.Println("================================================")

	engine.Shutdown()
}

func initDB() *gorm.DB {
	dbPath := filepath.Join(os.TempDir(), "qim-bench.db")
	os.Remove(dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
	}
	for _, pragma := range pragmas {
		if err := db.Exec(pragma).Error; err != nil {
			panic(err)
		}
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&dal.Conversation{},
		&dal.Member{},
		&dal.UserConversation{},
		&dal.User{},
		&dal.Message{},
		&dal.FriendRequest{},
		&dal.FriendGroup{},
		&dal.Friend{},
	); err != nil {
		panic(err)
	}

	return db
}

func initEngine() *actor.Engine {
	return actor.NewEngine(
		actor.WithMetrics(actor.NewDefaultMetrics()),
		actor.WithLogger(logx.NewActorLogger()),
	)
}

func initEventBus(engine *actor.Engine) eventbus.Bus {
	ref, err := engine.Spawn("eventbus", eventbus.NewEventBusActor())
	if err != nil {
		panic(err)
	}
	return eventbus.NewRefBus(ref)
}

func mustSpawn(engine *actor.Engine, name string, a actor.Actor) *actor.ActorRef {
	ref, err := engine.Spawn(name, a)
	if err != nil {
		panic(err)
	}
	return ref
}
