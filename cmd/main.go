package main

import (
	"fmt"
	"log"
	"net/http"

	"clearify/handler"
	"clearify/middleware"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/uploads", handlers.UploadHandler)
	mux.HandleFunc("/", handlers.ServerTemplate)

	handler := middleware.Logging(middleware.CorsMiddleware(mux))

	server := &http.Server{
		Addr:    ":8000",
		Handler: handler,
	}

	fmt.Println("Server running on : localhost:8080")
	log.Fatal(server.ListenAndServe())
}