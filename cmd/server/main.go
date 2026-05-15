package main

import (
	"log"

	"qim/internal/transport"
)

func main() {
	engine := initEngine()
	db := initDB()
	stores := initStores(db)
	initActors(engine, stores)
	handlers := initHandlers(engine, stores)
	dispatcher := initDispatcher(engine, stores)

	srv := transport.NewServer(engine, handlers.conv, handlers.user, handlers.msg, handlers.friend, dispatcher)

	log.Println("QIM server starting on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
