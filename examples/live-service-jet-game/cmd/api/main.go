package main

import (
	"log"
	"net/http"
	"os"

	"example.com/orbit-snake/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	server := api.NewServer(api.NewStore())
	log.Printf("api listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, server.Routes()))
}
