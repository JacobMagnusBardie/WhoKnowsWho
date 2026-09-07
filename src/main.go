package main

import (
	"log"
	"net/http"
	"os"

	"github.com/JacobMagnusBardie/WhoKnowsWho/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := server.NewRouter()

	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
