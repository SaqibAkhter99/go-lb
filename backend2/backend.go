package main

import (
	"fmt"
	"log"
	"net/http"
)

func backed_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8") // normal header
	w.WriteHeader(http.StatusOK)
	fmt.Printf("Received request from %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "You are connected to Server 1")
	fmt.Printf("Host: %s\n", r.Host)
	fmt.Printf("User-Agent: %s\n", r.UserAgent())
	fmt.Printf("Accept %s\n", r.Header.Get("Accept"))

}

func main() {
	http.HandleFunc("/", backed_handler)
	fmt.Println("Starting server at port 8081")
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal("Server failed:", err)
	}

}
