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
	backendAddr  string
	backendAddr2 string
}

func main() {
	var address bckAddrs
	address.backendAddr = "localhost:8080"
	address.backendAddr2 = "localhost:8081"
	requestChannel := make(chan *http.Request)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Printf("Load balancer started, listening on %s\n", listenAddr)

	// Start a goroutine to forward requests to the backend
	go forwardToBackend(requestChannel, address)

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

func forwardToBackend(requestChannel chan *http.Request, address bckAddrs) {
	client := &http.Client{}
	for request := range requestChannel {
		// Forward the request to the backend server
		backend := address.backendAddr
		request.RequestURI = ""
		request.URL.Host = backend
		request.URL.Scheme = "http"
		request.Host = backend

		fmt.Printf("Host: %s\n", request.Host)
		fmt.Printf("User-Agent: %s\n", request.UserAgent())
		fmt.Printf("Accept: %s\n", request.Header.Get("Accept"))

		// Forwarding request to backend server
		resp, err := client.Do(request)
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

		// Write the response back to the original client connection
		// Assuming conn is passed to this function (needs modification to the handleConnection)
		// conn.Write([]byte(resp.Status + "\n"))
		// conn.Write(body)

		// For simplicity, printing out the response
		fmt.Printf("Response Body: %s\n", body)
		fmt.Printf("Response from server: %s\n", resp.Status)
	}
}
