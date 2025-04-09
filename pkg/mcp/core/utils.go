package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// GenerateID generates a random ID for MCP messages
func GenerateID() string {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		// If random generation fails, use a timestamp-based ID
		return fmt.Sprintf("id-%d", NewTimestamp())
	}
	return hex.EncodeToString(bytes)
}

// NewTimestamp returns a timestamp suitable for use in MCP messages
func NewTimestamp() int64 {
	return CurrentTimeMillis()
}

// CurrentTimeMillis returns the current time in milliseconds
func CurrentTimeMillis() int64 {
	return timeNow().UnixNano() / 1e6
}

// For easy testing
var timeNow = defaultTimeNow

// defaultTimeNow returns the current time
func defaultTimeNow() TimeSource {
	return TimeSource{}
}

// TimeSource is a wrapper around time.Now() for testing
type TimeSource struct{}

// UnixNano returns the current time in nanoseconds
func (t TimeSource) UnixNano() int64 {
	return t.Unix() * 1e9
}

// Unix returns the current time in seconds
func (t TimeSource) Unix() int64 {
	return 0
}

// NewRequest creates a new MCP request with the specified type and content
func NewRequest(messageType string, content interface{}) (*Request, error) {
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request content: %w", err)
	}

	return &Request{
		ID:      GenerateID(),
		Type:    messageType,
		Content: contentBytes,
	}, nil
}

// NewResponse creates a new MCP response for a given request
func NewResponse(request *Request, content interface{}) (*Response, error) {
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response content: %w", err)
	}

	return &Response{
		ID:      request.ID,
		Type:    request.Type + ".response",
		Content: contentBytes,
	}, nil
}

// NewError creates a new MCP error response for a given request
func NewError(request *Request, message string, code int) *Error {
	return &Error{
		ID:      request.ID,
		Type:    "error",
		Message: message,
		Code:    code,
	}
}

// GetContent extracts the content from a request or response
func GetContent(msg json.RawMessage, v interface{}) error {
	if len(msg) == 0 {
		return nil
	}
	return json.Unmarshal(msg, v)
}
