package stdio

import (
	"context"
	"io"

	"github.com/hewenyu/go-mcp/pkg/mcp/core"
)

// Server implements the MCP server using STDIO transport
type Server struct {
	*core.BaseServer
	transport *Transport
}

// NewServer creates a new STDIO MCP server
func NewServer() *Server {
	transport := NewTransport()
	baseServer := core.NewBaseServer(transport, core.DefaultServerConfig)
	return &Server{
		BaseServer: baseServer,
		transport:  transport,
	}
}

// NewCustomServer creates a new STDIO MCP server with custom reader and writer
func NewCustomServer(reader io.Reader, writer io.Writer, config core.ServerConfig) *Server {
	transport := NewCustomTransport(reader, writer)
	baseServer := core.NewBaseServer(transport, config)
	return &Server{
		BaseServer: baseServer,
		transport:  transport,
	}
}

// RegisterHandler registers a message handler for a specific message type
func (s *Server) RegisterHandler(messageType string, handler core.Handler) {
	s.BaseServer.RegisterHandler(messageType, handler)
}

// SetDefaultHandler sets a default handler for message types that don't have a specific handler
func (s *Server) SetDefaultHandler(handler core.Handler) {
	s.BaseServer.SetDefaultHandler(handler)
}

// Start initializes the server and starts listening for messages
func (s *Server) Start(ctx context.Context) error {
	return s.BaseServer.Start(ctx)
}

// Stop stops the server
func (s *Server) Stop() error {
	return s.BaseServer.Stop()
}
