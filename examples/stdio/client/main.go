package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hewenyu/go-mcp/pkg/mcp/core"
	"github.com/hewenyu/go-mcp/pkg/mcp/stdio"
)

// EchoRequest represents the content of an echo request
type EchoRequest struct {
	Message string `json:"message"`
}

// EchoResponse represents the content of an echo response
type EchoResponse struct {
	Message string `json:"message"`
}

func main() {
	// Create a new STDIO client
	client := stdio.NewClient()

	// Start the client
	if err := client.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create an echo request
	echoReq := EchoRequest{
		Message: "Hello, MCP!",
	}

	// Create the MCP request
	req, err := core.NewRequest("echo", echoReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sending echo request...")

	// Send the request and wait for response
	resp, err := client.Send(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send request: %v\n", err)
		os.Exit(1)
	}

	// Parse the response
	var echoResp EchoResponse
	if err := core.GetContent(resp.Content, &echoResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	// Print the response
	fmt.Printf("Received response: %s\n", echoResp.Message)
}
