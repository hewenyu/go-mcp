package sse

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Transport implements the core.Transport interface using Server-Sent Events (SSE)
type Transport struct {
	client      *http.Client
	request     *http.Request
	writer      http.ResponseWriter
	url         string
	eventCh     chan []byte
	closeCh     chan struct{}
	mutex       sync.Mutex
	closeOnce   sync.Once
	initialized bool
	isServer    bool
	connections map[string]http.ResponseWriter
	connMutex   sync.RWMutex
	wg          sync.WaitGroup
}

// NewClientTransport creates a new SSE transport for a client
func NewClientTransport(url string) *Transport {
	return &Transport{
		client:      &http.Client{},
		url:         url,
		eventCh:     make(chan []byte),
		closeCh:     make(chan struct{}),
		connections: make(map[string]http.ResponseWriter),
		isServer:    false,
	}
}

// NewServerTransport creates a new SSE transport for a server
func NewServerTransport() *Transport {
	return &Transport{
		eventCh:     make(chan []byte),
		closeCh:     make(chan struct{}),
		connections: make(map[string]http.ResponseWriter),
		isServer:    true,
	}
}

// Initialize initializes the transport
func (t *Transport) Initialize() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.initialized {
		return nil
	}

	if !t.isServer {
		// Client mode: create HTTP request and start connection
		req, err := http.NewRequest(http.MethodGet, t.url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		t.request = req
		t.wg.Add(1)
		go t.connectToSSE()
	}

	t.initialized = true
	return nil
}

// connectToSSE connects to the SSE server and reads events
func (t *Transport) connectToSSE() {
	defer t.wg.Done()

	for {
		select {
		case <-t.closeCh:
			return
		default:
			// Connect to SSE endpoint
			resp, err := t.client.Do(t.request)
			if err != nil {
				// TODO: add backoff retry logic
				continue
			}

			// Process the SSE stream
			t.processSSEStream(resp.Body)

			// If we reach here, the connection was closed
			resp.Body.Close()

			// Check if we should exit
			select {
			case <-t.closeCh:
				return
			default:
				// Continue and try to reconnect
			}
		}
	}
}

// processSSEStream processes the SSE event stream
func (t *Transport) processSSEStream(body io.ReadCloser) {
	scanner := bufio.NewScanner(body)
	var buffer bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		// Empty line marks the end of an event
		if line == "" {
			if buffer.Len() > 0 {
				data := buffer.String()
				if strings.HasPrefix(data, "data: ") {
					data = strings.TrimPrefix(data, "data: ")
					select {
					case <-t.closeCh:
						return
					case t.eventCh <- []byte(data):
						// Event sent
					}
				}
				buffer.Reset()
			}
			continue
		}

		// Process event fields
		if strings.HasPrefix(line, "data: ") {
			if buffer.Len() > 0 {
				buffer.WriteRune('\n')
			}
			buffer.WriteString(line)
		}
		// We can add support for other event fields like 'event:' if needed
	}
}

// RegisterConnection registers a new SSE connection (server-side only)
func (t *Transport) RegisterConnection(id string, w http.ResponseWriter) {
	if !t.isServer {
		return
	}

	t.connMutex.Lock()
	defer t.connMutex.Unlock()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Store the connection
	t.connections[id] = w

	// Notify the client that the connection is established
	fmt.Fprint(w, "event: connected\ndata: {\"connected\":true}\n\n")
	w.(http.Flusher).Flush()
}

// RemoveConnection removes an SSE connection (server-side only)
func (t *Transport) RemoveConnection(id string) {
	if !t.isServer {
		return
	}

	t.connMutex.Lock()
	defer t.connMutex.Unlock()

	delete(t.connections, id)
}

// Send sends a message through the transport
func (t *Transport) Send(ctx context.Context, message []byte) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.initialized {
		return fmt.Errorf("transport not initialized")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if t.isServer {
			// Server mode: send to all connections
			t.connMutex.RLock()
			defer t.connMutex.RUnlock()

			for _, w := range t.connections {
				fmt.Fprintf(w, "data: %s\n\n", message)
				w.(http.Flusher).Flush()
			}
			return nil
		}

		// Client mode: POST data to server
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(message))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := t.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("server returned error status: %d", resp.StatusCode)
		}

		return nil
	}
}

// Receive returns a channel that receives messages
func (t *Transport) Receive(ctx context.Context) (<-chan []byte, error) {
	if !t.initialized {
		return nil, fmt.Errorf("transport not initialized")
	}

	// Create a new channel for the context
	ctxCh := make(chan []byte)

	// Forward messages from the main channel to the context channel
	go func() {
		defer close(ctxCh)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-t.eventCh:
				if !ok {
					return
				}
				select {
				case ctxCh <- msg:
					// Message forwarded
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ctxCh, nil
}

// Close closes the transport
func (t *Transport) Close() error {
	var err error

	t.closeOnce.Do(func() {
		close(t.closeCh)
		t.wg.Wait()

		t.mutex.Lock()
		t.initialized = false
		t.mutex.Unlock()

		if t.isServer {
			t.connMutex.Lock()
			t.connections = make(map[string]http.ResponseWriter)
			t.connMutex.Unlock()
		}
	})

	return err
}

// SSEHandler returns an HTTP handler for SSE connections
func (t *Transport) SSEHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// SSE connection request
			connID := r.URL.Query().Get("id")
			if connID == "" {
				connID = fmt.Sprintf("conn-%d", connIDCounter.Add(1))
			}

			t.RegisterConnection(connID, w)

			// Keep the connection open until client disconnects
			<-r.Context().Done()

			t.RemoveConnection(connID)
		} else if r.Method == http.MethodPost {
			// Message from client
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			select {
			case t.eventCh <- body:
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

// Atomic counter for connection IDs
type atomicCounter struct {
	value int64
	mutex sync.Mutex
}

func (c *atomicCounter) Add(delta int64) int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value += delta
	return c.value
}

var connIDCounter = atomicCounter{}
