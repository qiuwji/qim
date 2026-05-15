package main

import (
	"log"

	"qim/internal/actor"
	"qim/internal/gateway"
)

func main() {
	engine := actor.NewEngine(
		actor.WithMetrics(actor.NewDefaultMetrics()),
	)

	srv := gateway.NewServer(engine)

	log.Println("QIM server starting on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
