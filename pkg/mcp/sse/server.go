package sse

import (
	"context"
	"net/http"

	"github.com/hewenyu/go-mcp/pkg/mcp/core"
)

// Server implements the MCP server using SSE transport
type Server struct {
	*core.BaseServer
	transport *Transport
	http      *http.Server
}

// ServerConfig contains configuration options for an SSE MCP server
type ServerConfig struct {
	// Core server configuration
	Core core.ServerConfig

	// HTTP server address (e.g., ":8080")
	Address string

	// Path for SSE endpoint (e.g., "/events")
	Path string
}

// DefaultServerConfig provides default configuration values for an SSE server
var DefaultServerConfig = ServerConfig{
	Core:    core.DefaultServerConfig,
	Address: ":8080",
	Path:    "/events",
}

// NewServer creates a new SSE MCP server with default configuration
func NewServer() *Server {
	return NewServerWithConfig(DefaultServerConfig)
}

// NewServerWithConfig creates a new SSE MCP server with custom configuration
func NewServerWithConfig(config ServerConfig) *Server {
	transport := NewServerTransport()
	baseServer := core.NewBaseServer(transport, config.Core)

	// Create HTTP server
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:    config.Address,
		Handler: mux,
	}

	server := &Server{
		BaseServer: baseServer,
		transport:  transport,
		http:       httpServer,
	}

	// Register SSE handler
	mux.Handle(config.Path, transport.SSEHandler())

	return server
}

// RegisterHandler registers a message handler for a specific message type
func (s *Server) RegisterHandler(messageType string, handler core.Handler) {
	s.BaseServer.RegisterHandler(messageType, handler)
}

// SetDefaultHandler sets a default handler for message types that don't have a specific handler
func (s *Server) SetDefaultHandler(handler core.Handler) {
	s.BaseServer.SetDefaultHandler(handler)
}

// Start initializes the server and starts listening for HTTP connections
func (s *Server) Start(ctx context.Context) error {
	// Start the base server
	if err := s.BaseServer.Start(ctx); err != nil {
		return err
	}

	// Start HTTP server in a separate goroutine
	go func() {
		// Ignore error as it will be returned when the server is stopped
		s.http.ListenAndServe()
	}()

	return nil
}

// Stop stops the server and closes all connections
func (s *Server) Stop() error {
	// Shutdown HTTP server
	s.http.Shutdown(context.Background())

	// Stop the base server
	return s.BaseServer.Stop()
}

// AddCustomHandler adds a custom HTTP handler to the server
func (s *Server) AddCustomHandler(pattern string, handler http.Handler) {
	mux := s.http.Handler.(*http.ServeMux)
	mux.Handle(pattern, handler)
}
