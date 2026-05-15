package main

import (
	"log"

	"qim/internal/transport"
)

func main() {
	engine := initEngine()
	db := initDB()
	stores := initStores(db)
	svcs := initServices(engine, stores)
	initActors(engine, stores)
	handlers := initHandlers(svcs)
	dispatcher := initDispatcher(svcs)

	srv := transport.NewServer(engine, handlers, dispatcher)

	log.Println("QIM server starting on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
