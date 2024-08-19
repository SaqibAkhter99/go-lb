package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

var listenAddr = ":80"

type bckAddrs struct {
	backendAddr string
	value       string
	healthy     bool
}

func main() {
	address := []bckAddrs{
		{"backendAddr", "localhost:8080", true},
		{"backendAddr2", "localhost:8081", true},
	}
	servCount := len(address)
	requestChannel := make(chan *http.Request)
	go monitorServers(address) // Monitor servers in the background
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

func monitorServers(address []bckAddrs) {
	for {
		for i := range address {
			hcUrl := "http://" + address[i].value + "/hc"
			_, err := http.Get(hcUrl)
			if err != nil {
				address[i].healthy = false
				fmt.Fprintf(os.Stderr, "Backend Server %d (%s) is down: %v\n", i, address[i].value, err)
			} else {
				address[i].healthy = true
				//fmt.Printf("Backend Server %d (%s) is healthy\n", i, address[i].value)
			}
		}
		time.Sleep(10 * time.Second) // Check health every 10 seconds
	}
}

func roundRobbin(address []bckAddrs, currPos int) (string, int, bool) {
	// Loop through all servers to find the next healthy one
	for i := 0; i < len(address); i++ {
		currPos = (currPos + 1) % len(address)
		if address[currPos].healthy {
			return address[currPos].value, currPos, true
		}
	}
	// If no healthy server is found
	return "", currPos, false
}

func forwardToBackend(requestChannel chan *http.Request, address []bckAddrs, servCount int) {
	var currPos int
	client := &http.Client{}

	for request := range requestChannel {
		currServer, newPos, found := roundRobbin(address, currPos)
		currPos = newPos

		if !found {
			// No healthy server available
			fmt.Fprintf(os.Stderr, "No healthy backend servers available\n")
			continue
		}

		// Copy the request body
		var requestBody io.ReadCloser
		if request.Body != nil {
			buf, err := io.ReadAll(request.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
				continue
			}
			requestBody = io.NopCloser(bytes.NewBuffer(buf))
			// Reset the request body for the original request
			request.Body = io.NopCloser(bytes.NewBuffer(buf))
		}

		// Create a new request for the backend server
		newReq, err := http.NewRequest(request.Method, "http://"+currServer+request.URL.Path, requestBody)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating new request: %v\n", err)
			continue
		}
		newReq.Header = request.Header
		newReq.Host = currServer

		// Send the request to the backend server
		resp, err := client.Do(newReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forwarding request to backend server %s: %v\n", currServer, err)
			address[currPos].healthy = false // Mark server as unhealthy
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
