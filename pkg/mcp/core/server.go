package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ServerConfig contains configuration options for an MCP server
type ServerConfig struct {
	// Additional server configuration options can be added here
}

// DefaultServerConfig represents the default configuration for an MCP server
var DefaultServerConfig = ServerConfig{}

// BaseServer provides the basic functionality for an MCP server
type BaseServer struct {
	transport      Transport
	config         ServerConfig
	handlers       map[string]Handler
	defaultHandler Handler
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewBaseServer creates a new BaseServer with the given transport and configuration
func NewBaseServer(transport Transport, config ServerConfig) *BaseServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseServer{
		transport: transport,
		config:    config,
		handlers:  make(map[string]Handler),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RegisterHandler registers a message handler for a specific message type
func (s *BaseServer) RegisterHandler(messageType string, handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handlers[messageType] = handler
}

// SetDefaultHandler sets a default handler for message types that don't have a specific handler
func (s *BaseServer) SetDefaultHandler(handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.defaultHandler = handler
}

// Start initializes the server and starts listening for messages
func (s *BaseServer) Start(ctx context.Context) error {
	if err := s.transport.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize transport: %w", err)
	}

	s.wg.Add(1)
	go s.receiveLoop(ctx)

	return nil
}

// receiveLoop continuously receives messages and routes them to the appropriate handlers
func (s *BaseServer) receiveLoop(ctx context.Context) {
	defer s.wg.Done()

	// Create a merged context that can be canceled by either s.ctx or the provided ctx
	mergedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-s.ctx.Done():
			cancel()
		case <-ctx.Done():
			// The context provided to Start() has been canceled
		}
	}()

	msgCh, err := s.transport.Receive(mergedCtx)
	if err != nil {
		// Log error or handle it appropriately
		return
	}

	for {
		select {
		case <-mergedCtx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}

			// Handle the message in a separate goroutine
			s.wg.Add(1)
			go func(message []byte) {
				defer s.wg.Done()
				s.handleMessage(mergedCtx, message)
			}(msg)
		}
	}
}

// handleMessage processes an incoming message and routes it to the appropriate handler
func (s *BaseServer) handleMessage(ctx context.Context, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		// Not a valid message, ignore
		return
	}

	var handler Handler
	var found bool

	s.mutex.RLock()
	handler, found = s.handlers[msg.Type]
	if !found {
		handler = s.defaultHandler
	}
	s.mutex.RUnlock()

	if handler == nil {
		// No handler available
		if msg.Type != "" {
			// If it looks like a request, send an error response
			var req Request
			if err := json.Unmarshal(data, &req); err == nil && req.ID != "" {
				errResp := Error{
					ID:      req.ID,
					Type:    "error",
					Message: fmt.Sprintf("Unsupported message type: %s", msg.Type),
					Code:    404,
				}
				respData, _ := json.Marshal(errResp)
				s.transport.Send(ctx, respData)
			}
		}
		return
	}

	// Process the message with the appropriate handler
	respData, err := handler.HandleMessage(ctx, data)
	if err != nil {
		// Handle error (possibly by sending an error response)
		var req Request
		if err := json.Unmarshal(data, &req); err == nil && req.ID != "" {
			errResp := Error{
				ID:      req.ID,
				Type:    "error",
				Message: err.Error(),
				Code:    500,
			}
			respData, _ := json.Marshal(errResp)
			s.transport.Send(ctx, respData)
		}
		return
	}

	// If there's a response to send, send it
	if respData != nil {
		s.transport.Send(ctx, respData)
	}
}

// Stop stops the server and waits for all handlers to complete
func (s *BaseServer) Stop() error {
	s.cancel()
	s.wg.Wait()
	return s.transport.Close()
}
