package sse

import (
	"context"

	"github.com/hewenyu/go-mcp/pkg/mcp/core"
)

// Client implements the MCP client using SSE transport
type Client struct {
	*core.BaseClient
	transport *Transport
}

// NewClient creates a new SSE MCP client
func NewClient(serverURL string) *Client {
	transport := NewClientTransport(serverURL)
	baseClient := core.NewBaseClient(transport, core.DefaultClientConfig)
	return &Client{
		BaseClient: baseClient,
		transport:  transport,
	}
}

// NewClientWithConfig creates a new SSE MCP client with custom configuration
func NewClientWithConfig(serverURL string, config core.ClientConfig) *Client {
	transport := NewClientTransport(serverURL)
	baseClient := core.NewBaseClient(transport, config)
	return &Client{
		BaseClient: baseClient,
		transport:  transport,
	}
}

// Send sends a request and waits for a response
func (c *Client) Send(ctx context.Context, request *core.Request) (*core.Response, error) {
	return c.BaseClient.Send(ctx, request)
}

// Close closes the client and its transport
func (c *Client) Close() error {
	return c.BaseClient.Close()
}
