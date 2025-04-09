package core

import (
	"context"
	"encoding/json"
)

// Message represents a general MCP message
type Message struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

// Request represents an MCP request message
type Request struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

// Response represents an MCP response message
type Response struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

// Error represents an MCP error message
type Error struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Transport defines the interface for different transport mechanisms (STDIO, SSE, etc.)
type Transport interface {
	// Initialize initializes the transport
	Initialize() error

	// Send sends a message through the transport
	Send(ctx context.Context, message []byte) error

	// Receive returns a channel that receives messages
	Receive(ctx context.Context) (<-chan []byte, error)

	// Close closes the transport
	Close() error
}

// Handler defines the interface for handling MCP messages
type Handler interface {
	// HandleMessage processes an incoming MCP message
	HandleMessage(ctx context.Context, message []byte) ([]byte, error)
}

// Client defines the interface for an MCP client
type Client interface {
	// Send sends a request and waits for a response
	Send(ctx context.Context, request *Request) (*Response, error)

	// Close closes the client
	Close() error
}

// Server defines the interface for an MCP server
type Server interface {
	// Start starts the server
	Start(ctx context.Context) error

	// Stop stops the server
	Stop() error

	// RegisterHandler registers a message handler for a specific message type
	RegisterHandler(messageType string, handler Handler)
}
