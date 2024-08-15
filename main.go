package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
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
	// Get the type of the struct
	t := reflect.TypeOf(address)
	elemType := t.Elem()
	servCount := elemType.NumField()
	// Get the number of fields in the struct

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

func roundRobbin(requestChannel chan *http.Request, address []bckAddrs, currPos int64, servCount int64) string {
	var roundRobbin []string
	for _, val := range address {
		roundRobbin = append(roundRobbin, val.value)
	}
	fmt.Println(roundRobbin[currPos])
	return roundRobbin[currPos]
}

func forwardToBackend(requestChannel chan *http.Request, address []bckAddrs, servCount int) {
	var currPos int64
	client := &http.Client{}
	for request := range requestChannel {
		currServer := roundRobbin(requestChannel, address, currPos, int64(servCount))
		currPos += 1
		if currPos >= int64(servCount) {
			currPos = 0
		}
		fmt.Println(currPos)
		// Forward the request to the backend server
		backend := currServer
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
		// For simplicity, printing out the response
		fmt.Printf("Response Body: %s\n", body)
		fmt.Printf("Response from server: %s\n", resp.Status)
	}
}
