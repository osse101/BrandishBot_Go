package main

import (
	"log"

	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

func main() {
	userService := user.NewService()
	srv := server.NewServer(8080, userService)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
