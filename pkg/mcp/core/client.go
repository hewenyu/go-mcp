package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultClientConfig represents the default configuration for an MCP client
var DefaultClientConfig = ClientConfig{
	Timeout: 30 * time.Second,
}

// ClientConfig contains configuration options for an MCP client
type ClientConfig struct {
	// Timeout is the default timeout for requests
	Timeout time.Duration
}

// BaseClient provides the basic functionality for an MCP client
type BaseClient struct {
	transport Transport
	config    ClientConfig
	pending   map[string]chan *Response
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewBaseClient creates a new BaseClient with the given transport and configuration
func NewBaseClient(transport Transport, config ClientConfig) *BaseClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseClient{
		transport: transport,
		config:    config,
		pending:   make(map[string]chan *Response),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start initializes the client and starts listening for messages
func (c *BaseClient) Start() error {
	if err := c.transport.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize transport: %w", err)
	}

	c.wg.Add(1)
	go c.receiveLoop()

	return nil
}

// receiveLoop continuously receives messages and routes them to the appropriate handlers
func (c *BaseClient) receiveLoop() {
	defer c.wg.Done()

	msgCh, err := c.transport.Receive(c.ctx)
	if err != nil {
		// Log error or handle it appropriately
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			c.handleResponse(msg)
		}
	}
}

// handleResponse processes a response message and routes it to the appropriate pending request
func (c *BaseClient) handleResponse(data []byte) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		// Could be an error message
		var errResp Error
		if err := json.Unmarshal(data, &errResp); err != nil {
			// Not a valid response or error, ignore
			return
		}
		// Process error response
		c.mutex.RLock()
		ch, ok := c.pending[errResp.ID]
		c.mutex.RUnlock()
		if ok {
			// Convert error to response for the waiting goroutine
			errContent, _ := json.Marshal(errResp)
			errorResp := &Response{
				ID:      errResp.ID,
				Type:    "error",
				Content: errContent,
			}
			ch <- errorResp
		}
		return
	}

	c.mutex.RLock()
	ch, ok := c.pending[resp.ID]
	c.mutex.RUnlock()
	if ok {
		ch <- &resp
	}
}

// Send sends a request and waits for a response
func (c *BaseClient) Send(ctx context.Context, request *Request) (*Response, error) {
	respCh := make(chan *Response, 1)

	c.mutex.Lock()
	c.pending[request.ID] = respCh
	c.mutex.Unlock()

	defer func() {
		c.mutex.Lock()
		delete(c.pending, request.ID)
		c.mutex.Unlock()
		close(respCh)
	}()

	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := c.transport.Send(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Use either the provided context or a timeout context
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()
	}

	select {
	case resp := <-respCh:
		if resp.Type == "error" {
			var errResp Error
			if err := json.Unmarshal(resp.Content, &errResp); err != nil {
				return nil, errors.New("unknown error")
			}
			return nil, fmt.Errorf("error code %d: %s", errResp.Code, errResp.Message)
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the client and its transport
func (c *BaseClient) Close() error {
	c.cancel()
	c.wg.Wait()
	return c.transport.Close()
}
