package main

import (
	"log"

	"github.com/osse101/BrandishBot_Go/internal/server"
)

func main() {
	srv := server.NewServer(8080)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
