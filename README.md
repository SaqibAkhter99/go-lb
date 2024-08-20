# Go HTTP Load Balancer

This project implements a simple HTTP load balancer in Go, which distributes incoming requests to multiple backend servers. The load balancer monitors the health of backend servers and routes requests to healthy servers using a round-robin algorithm.

## Project Structure

The project consists of the following components:

- **Load Balancer**: The main component that handles incoming requests and forwards them to backend servers.
- **Backend Servers**: Two backend servers, each running in a separate folder, which respond to requests and provide basic health check endpoints.

### Folder Structure
.
    ├── load-balancer/
    │ ├── main.go # Load balancer implementation
    ├── backend1/
    │ ├── main.go # Backend server 1 implementation
    ├── backend2/
    │ ├── main.go # Backend server 2 implementation
    └── README.md # This file


## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) 1.18 or later
- Basic knowledge of Go programming

### Running the Backend Servers

1. Navigate to the `backend1/` folder:
    ```bash
    cd backend1/
    ```
2. Start the first backend server:
    ```bash
    go run main.go
    ```
3. Navigate to the `backend2/` folder:
    ```bash
    cd ../backend2/
    ```
4. Start the second backend server:
    ```bash
    go run main.go
    ```

### Running the Load Balancer

1. Navigate to the `load-balancer/` folder:
    ```bash
    cd ../load-balancer/
    ```
2. Start the load balancer:
    ```bash
    go run main.go
    ```
3. The load balancer will start listening on port 80. You can send requests to it using `curl` or a web browser:
    ```bash
    curl http://localhost/
    ```

## Features

- **Round-Robin Load Balancing**: Distributes incoming requests evenly across backend servers.
- **Health Check Monitoring**: Continuously monitors the health of backend servers and avoids routing requests to unhealthy servers.
- **Dynamic Server Recovery**: Automatically redirects traffic to a server when it becomes healthy again after a failure.

## Troubleshooting

- **Connection Refused**: Ensure that both backend servers are running before starting the load balancer.
- **Empty Reply from Server**: This could indicate that all backend servers are down or the load balancer is unable to forward the request. Check the logs for more information.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request with any improvements or bug fixes.

## License

This project is licensed under the MIT License.
