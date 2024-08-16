package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

var listenAddr = ":80"

type bckAddrs struct {
	backendAddr string
	value       string
}

func main() {
	address := []bckAddrs{
		{"backendAddr", "localhost:8080"},
		{"backendAddr2", "localhost:8081"},
	}
	servCount := len(address)
	requestChannel := make(chan *http.Request)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Printf("Load balancer started, listening on %s\n", listenAddr)

	// Start a goroutine to forward requests to the backend
	go forwardToBackend(requestChannel, address, servCount)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue
		}
		// Handle connection and send request to channel
		go handleConnection(conn, requestChannel)
	}
}

func handleConnection(conn net.Conn, requestChannel chan *http.Request) {
	defer conn.Close()

	// Log incoming connection
	remoteAddr := conn.RemoteAddr().String()
	fmt.Printf("Received request from %s\n", remoteAddr)

	// Read the request
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request: %v\n", err)
		return
	}

	fmt.Printf("%s %s %s\n", request.Method, request.URL, request.Proto)

	// Send the request to the request channel
	requestChannel <- request
}

func roundRobbin(address []bckAddrs, currPos int) string {
	return address[currPos].value
}

func forwardToBackend(requestChannel chan *http.Request, address []bckAddrs, servCount int) {
	var currPos int
	client := &http.Client{}
	for request := range requestChannel {
		currServer := roundRobbin(address, currPos)
		currPos = (currPos + 1) % servCount

		// Create a new request to avoid reusing the original one
		newReq, err := http.NewRequest(request.Method, request.URL.String(), request.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating new request: %v\n", err)
			continue
		}
		newReq.Header = request.Header
		newReq.URL.Host = currServer
		newReq.URL.Scheme = "http"
		newReq.Host = currServer

		// Forwarding request to backend server
		resp, err := client.Do(newReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forwarding request: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// Log the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
			continue
		}
		// For simplicity, printing out the response
		fmt.Printf("Response Body: %s\n", body)
		fmt.Printf("Response from server: %s\n", resp.Status)
	}
}
