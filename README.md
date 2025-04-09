# Go-MCP: Model Context Protocol SDK for Go

Go-MCP is a Go SDK that implements the Model Context Protocol, allowing developers to quickly build applications that communicate with language models using both STDIO and Server-Sent Events (SSE).

## Features

- Support for both STDIO and SSE transport mechanisms
- Easy-to-use client and server implementations
- Extensible message handling system
- Asynchronous communication
- Reliable message delivery
- Type-safe message content handling

## Installation

```bash
go get github.com/hewenyu/go-mcp
```

## Quick Start

### STDIO Example

#### Server

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/hewenyu/go-mcp/pkg/mcp/core"
    "github.com/hewenyu/go-mcp/pkg/mcp/stdio"
)

// Define your message handler
type EchoHandler struct{}

func (h *EchoHandler) HandleMessage(ctx context.Context, data []byte) ([]byte, error) {
    // Parse the request
    var req core.Request
    if err := json.Unmarshal(data, &req); err != nil {
        return nil, err
    }

    // Process the request and generate a response
    // ...

    // Return the response
    resp, _ := core.NewResponse(&req, responseContent)
    respData, _ := json.Marshal(resp)
    return respData, nil
}

func main() {
    // Create a new STDIO server
    server := stdio.NewServer()

    // Register your message handler
    server.RegisterHandler("echo", &EchoHandler{})

    // Start the server
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    server.Start(ctx)

    // Wait for termination signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    // Stop the server
    server.Stop()
}
```

#### Client

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/hewenyu/go-mcp/pkg/mcp/core"
    "github.com/hewenyu/go-mcp/pkg/mcp/stdio"
)

func main() {
    // Create a new STDIO client
    client := stdio.NewClient()

    // Start the client
    client.Start()
    defer client.Close()

    // Create a request
    req, _ := core.NewRequest("echo", map[string]string{"message": "Hello, MCP!"})

    // Send the request and wait for response
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    resp, err := client.Send(ctx, req)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Process the response
    fmt.Printf("Response: %+v\n", resp)
}
```

### SSE Example

#### Server

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/hewenyu/go-mcp/pkg/mcp/core"
    "github.com/hewenyu/go-mcp/pkg/mcp/sse"
)

func main() {
    // Create server configuration
    config := sse.ServerConfig{
        Core:    core.DefaultServerConfig,
        Address: ":8080",
        Path:    "/events",
    }

    // Create a new SSE server
    server := sse.NewServerWithConfig(config)

    // Register your message handler
    server.RegisterHandler("echo", &EchoHandler{})

    // Start the server
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    server.Start(ctx)

    fmt.Println("Server started on http://localhost:8080/events")

    // Wait for termination signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    // Stop the server
    server.Stop()
}
```

#### Client

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/hewenyu/go-mcp/pkg/mcp/core"
    "github.com/hewenyu/go-mcp/pkg/mcp/sse"
)

func main() {
    // Create a new SSE client
    client := sse.NewClient("http://localhost:8080/events")

    // Start the client
    client.Start()
    defer client.Close()

    // Create a request
    req, _ := core.NewRequest("echo", map[string]string{"message": "Hello, MCP!"})

    // Send the request and wait for response
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    resp, err := client.Send(ctx, req)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Process the response
    fmt.Printf("Response: %+v\n", resp)
}
```

## Custom Message Handlers

You can create custom message handlers by implementing the `core.Handler` interface:

```go
type MyHandler struct{}

func (h *MyHandler) HandleMessage(ctx context.Context, data []byte) ([]byte, error) {
    // Parse the request
    var req core.Request
    if err := json.Unmarshal(data, &req); err != nil {
        return nil, err
    }

    // Extract request content
    var myReq MyRequestType
    if err := core.GetContent(req.Content, &myReq); err != nil {
        return nil, err
    }

    // Process the request and generate a response
    myResp := MyResponseType{
        // ...
    }

    // Create and return the response
    resp, err := core.NewResponse(&req, myResp)
    if err != nil {
        return nil, err
    }

    respData, err := json.Marshal(resp)
    if err != nil {
        return nil, err
    }

    return respData, nil
}
```

## Examples

Check out the `examples` directory for complete working examples:

- STDIO examples:
  - `examples/stdio/server` - STDIO server example
  - `examples/stdio/client` - STDIO client example

- SSE examples:
  - `examples/sse/server` - SSE server example
  - `examples/sse/client` - SSE client example

## License

This project is licensed under the [MIT License](LICENSE). 