package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
)

const (
	listenAddr  = ":80"
	backendAddr = "localhost:8080"
)

func main() {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Printf("Load balancer started, listening on %s\n", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
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

	// Forward the request to the backend server
	client := &http.Client{}
	request.RequestURI = ""
	request.URL.Host = backendAddr
	request.URL.Scheme = "http"
	request.Host = backendAddr

	resp, err := client.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error forwarding request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	//Write the response back to the client
	if err := resp.Write(conn); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
	}
}
