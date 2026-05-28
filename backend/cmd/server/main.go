package main

import (
	"qim/internal/pkg/logx"
	"qim/internal/transport"
	"qim/internal/transport/agent"

	"go.uber.org/zap"
)

func main() {
	logx.Init()
	defer zap.L().Sync()

	engine := initEngine()
	events := initEventBus(engine)
	db := initDB()
	stores := initStores(db)
	svcs := initServices(engine, stores, events)
	jwt := initJWT()
	initActors(engine, stores, events)
	initEventHandlers(engine, events)
	handlers := initHandlers(svcs, jwt, stores)
	approvals := agent.NewApprovalManager(stores.agent)
	dispatcher := initDispatcher(svcs, engine, approvals)
	agentDispatcher := initAgentDispatcher(svcs, stores, handlers, engine, events, approvals)

	srv := transport.NewServer(engine, handlers, events, dispatcher, jwt, agentDispatcher)

	zap.L().Info("QIM server starting", zap.String("addr", ":8080"))
	if err := srv.Run(":8080"); err != nil {
		zap.L().Fatal("QIM server stopped", zap.Error(err))
	}
}
