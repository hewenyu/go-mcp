package stdio

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// Transport implements the core.Transport interface using STDIO
type Transport struct {
	reader      *bufio.Reader
	writer      io.Writer
	initialized bool
	mutex       sync.Mutex
	msgCh       chan []byte
	closeCh     chan struct{}
	wg          sync.WaitGroup
}

// NewTransport creates a new STDIO transport
func NewTransport() *Transport {
	return &Transport{
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
		msgCh:   make(chan []byte),
		closeCh: make(chan struct{}),
	}
}

// NewCustomTransport creates a new STDIO transport with custom reader and writer
func NewCustomTransport(reader io.Reader, writer io.Writer) *Transport {
	return &Transport{
		reader:  bufio.NewReader(reader),
		writer:  writer,
		msgCh:   make(chan []byte),
		closeCh: make(chan struct{}),
	}
}

// Initialize initializes the transport
func (t *Transport) Initialize() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.initialized {
		return nil
	}

	t.wg.Add(1)
	go t.readLoop()

	t.initialized = true
	return nil
}

// readLoop continuously reads messages from stdin
func (t *Transport) readLoop() {
	defer t.wg.Done()

	scanner := bufio.NewScanner(t.reader)
	for scanner.Scan() {
		select {
		case <-t.closeCh:
			return
		default:
			line := scanner.Bytes()
			data := make([]byte, len(line))
			copy(data, line)
			t.msgCh <- data
		}
	}

	// If scanner.Scan() returns false, close the message channel
	close(t.msgCh)
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
		_, err := fmt.Fprintln(t.writer, string(message))
		return err
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
			case msg, ok := <-t.msgCh:
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
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.initialized {
		return nil
	}

	close(t.closeCh)
	t.wg.Wait()
	t.initialized = false

	return nil
}
