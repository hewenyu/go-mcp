package stdio

import (
	"context"
	"io"

	"github.com/hewenyu/go-mcp/pkg/mcp/core"
)

// Client implements the MCP client using STDIO transport
type Client struct {
	*core.BaseClient
	transport *Transport
}

// NewClient creates a new STDIO MCP client
func NewClient() *Client {
	transport := NewTransport()
	baseClient := core.NewBaseClient(transport, core.DefaultClientConfig)
	return &Client{
		BaseClient: baseClient,
		transport:  transport,
	}
}

// NewCustomClient creates a new STDIO MCP client with custom reader and writer
func NewCustomClient(reader io.Reader, writer io.Writer, config core.ClientConfig) *Client {
	transport := NewCustomTransport(reader, writer)
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
