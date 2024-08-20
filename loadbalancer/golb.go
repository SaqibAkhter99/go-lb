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

type ClientRequest struct {
	RespWriter *CustomResponseWriter
	Req        *http.Request
}

func main() {
	address := []bckAddrs{
		{"backendAddr", "localhost:8080", true},
		{"backendAddr2", "localhost:8081", true},
	}
	servCount := len(address)
	requestChannel := make(chan *ClientRequest)
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

func handleConnection(conn net.Conn, requestChannel chan *ClientRequest) {
	// Log incoming connection
	remoteAddr := conn.RemoteAddr().String()
	fmt.Printf("Received request from %s\n", remoteAddr)

	// Read the request
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request: %v\n", err)
		conn.Close()
		return
	}

	fmt.Printf("%s %s %s\n", request.Method, request.URL, request.Proto)

	// Create a custom response writer and send it along with the request
	respWriter := NewCustomResponseWriter(conn)

	// Send the request to the request channel
	requestChannel <- &ClientRequest{
		RespWriter: respWriter,
		Req:        request,
	}

	// Wait for the response to be fully written
	<-respWriter.finished
	conn.Close() // Close the connection after the response is sent
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

func forwardToBackend(requestChannel chan *ClientRequest, address []bckAddrs, servCount int) {
	var currPos int
	client := &http.Client{}

	for clientReq := range requestChannel {
		request := clientReq.Req
		respWriter := clientReq.RespWriter

		currServer, newPos, found := roundRobbin(address, currPos)
		currPos = newPos

		if !found {
			// No healthy server available
			fmt.Fprintf(os.Stderr, "No healthy backend servers available\n")
			http.Error(respWriter, "No healthy backend servers available", http.StatusServiceUnavailable)
			respWriter.Finish()
			continue
		}

		// Copy the request body
		var requestBody io.ReadCloser
		if request.Body != nil {
			buf, err := io.ReadAll(request.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
				http.Error(respWriter, "Error reading request body", http.StatusInternalServerError)
				respWriter.Finish()
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
			http.Error(respWriter, "Error creating new request", http.StatusInternalServerError)
			respWriter.Finish()
			continue
		}
		newReq.Header = request.Header
		newReq.Host = currServer

		// Send the request to the backend server
		resp, err := client.Do(newReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forwarding request to backend server %s: %v\n", currServer, err)
			address[currPos].healthy = false // Mark server as unhealthy
			http.Error(respWriter, "Error forwarding request to backend server", http.StatusBadGateway)
			respWriter.Finish()
			continue
		}
		defer resp.Body.Close()

		// Copy the response headers
		for key, values := range resp.Header {
			for _, value := range values {
				respWriter.Header().Add(key, value)
			}
		}

		// Write the status code to the response writer
		respWriter.WriteHeader(resp.StatusCode)

		// Copy the response body to the response writer
		if _, err := io.Copy(respWriter, resp.Body); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying response body to client: %v\n", err)
			http.Error(respWriter, "Error copying response body", http.StatusInternalServerError)
		}

		fmt.Printf("Response from server: %s\n", resp.Status)
		respWriter.Finish() // Indicate that the response has been fully written
	}
}

type CustomResponseWriter struct {
	conn     net.Conn
	header   http.Header
	status   int
	finished chan struct{} // To signal when writing is complete
}

func NewCustomResponseWriter(conn net.Conn) *CustomResponseWriter {
	return &CustomResponseWriter{
		conn:     conn,
		header:   make(http.Header),
		status:   http.StatusOK,
		finished: make(chan struct{}),
	}
}

func (w *CustomResponseWriter) Header() http.Header {
	return w.header
}

func (w *CustomResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(data)
}

func (w *CustomResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))
	w.conn.Write([]byte(statusLine))

	for k, v := range w.header {
		for _, vv := range v {
			headerLine := fmt.Sprintf("%s: %s\r\n", k, vv)
			w.conn.Write([]byte(headerLine))
		}
	}

	w.conn.Write([]byte("\r\n")) // End of headers
}

func (w *CustomResponseWriter) Finish() {
	close(w.finished)
}
