package main

import (
	"qim/internal/pkg/logx"
	"qim/internal/transport"

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
	handlers := initHandlers(svcs, jwt)
	dispatcher := initDispatcher(svcs, engine)

	srv := transport.NewServer(engine, handlers, events, dispatcher, jwt)

	zap.L().Info("QIM server starting", zap.String("addr", ":8080"))
	if err := srv.Run(":8080"); err != nil {
		zap.L().Fatal("QIM server stopped", zap.Error(err))
	}
}
