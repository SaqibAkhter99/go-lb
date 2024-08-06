package main

import (
	"fmt"
	"log"
	"net/http"
)

func backed_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "You are connected to Server 1")
}

func main() {
	http.HandleFunc("/", backed_handler)
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
