package main

import (
	"log"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/app"
)

func main() {
	server := app.NewServer()

	log.Printf("API server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
