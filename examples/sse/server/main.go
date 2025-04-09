package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hewenyu/go-mcp/pkg/mcp/core"
	"github.com/hewenyu/go-mcp/pkg/mcp/sse"
)

// EchoHandler handles echo requests
type EchoHandler struct{}

// EchoRequest represents the content of an echo request
type EchoRequest struct {
	Message string `json:"message"`
}

// EchoResponse represents the content of an echo response
type EchoResponse struct {
	Message string `json:"message"`
}

// HandleMessage processes an incoming echo message
func (h *EchoHandler) HandleMessage(ctx context.Context, data []byte) ([]byte, error) {
	var req core.Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Parse the request content
	var echoReq EchoRequest
	if err := core.GetContent(req.Content, &echoReq); err != nil {
		return nil, fmt.Errorf("failed to parse echo request: %w", err)
	}

	// Create the response content
	echoResp := EchoResponse{
		Message: fmt.Sprintf("Echo: %s", echoReq.Message),
	}

	// Create and marshal the response
	resp, err := core.NewResponse(&req, echoResp)
	if err != nil {
		return nil, fmt.Errorf("failed to create response: %w", err)
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return respData, nil
}

func main() {
	// Create a server configuration
	config := sse.ServerConfig{
		Core:    core.DefaultServerConfig,
		Address: ":8080",
		Path:    "/events",
	}

	// Create a new SSE server
	server := sse.NewServerWithConfig(config)

	// Register the echo handler
	server.RegisterHandler("echo", &EchoHandler{})

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("MCP SSE Server started on http://localhost:8080/events")
	fmt.Println("Press Ctrl+C to exit.")

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")

	// Stop the server
	if err := server.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
	}
}
