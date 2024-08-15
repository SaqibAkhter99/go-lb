package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HealthCheckResponse struct {
	Status        string `json:"status"`
	TotalDuration string `json:"totalDuration"`
	Message       string `json:"serverResponse"`
}

func backed__handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8") // normal header
	w.WriteHeader(http.StatusOK)
	fmt.Printf("Received request from %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "You are connected to Server 2")
	fmt.Printf("Host: %s\n", r.Host)
	fmt.Printf("User-Agent: %s\n", r.UserAgent())
	fmt.Printf("Accept %s\n", r.Header.Get("Accept"))

}

func main() {
	http.HandleFunc("/", backed__handler)
	http.HandleFunc("/hc", backed__healthCheck__handler)
	fmt.Println("Starting server at port 8081")
	if err := http.ListenAndServe("localhost:8081", nil); err != nil {
		log.Fatal("Server failed:", err)
	}

}

func backed__healthCheck__handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Process the request
	// (Simulate some processing here if needed)

	// End timing
	elapsed := time.Since(start)

	// Format the response duration as a string
	formattedDuration := fmt.Sprintf("%02d:%02d:%02d.%06d",
		int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60, elapsed.Microseconds()%1000000)

	// Create the response object
	response := HealthCheckResponse{
		Status:        "Healthy",
		TotalDuration: formattedDuration,
		Message:       "Server 2 is runnimg",
	}

	// Convert the response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error generating JSON", http.StatusInternalServerError)
		return
	}

	// Set the content type to JSON and send the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)

	// Log the request details
	fmt.Printf("Received request from %s\n", r.RemoteAddr)
	fmt.Printf("Host: %s\n", r.Host)
	fmt.Printf("User-Agent: %s\n", r.UserAgent())
	fmt.Printf("Accept: %s\n", r.Header.Get("Accept"))
}
