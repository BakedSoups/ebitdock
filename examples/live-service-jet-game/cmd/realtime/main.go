package main

import (
	"log"
	"net/http"
	"os"

	"example.com/orbit-snake/internal/realtime"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}
	hub := realtime.NewHub()
	go hub.Run()
	log.Printf("realtime listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, hub.Routes()))
}
