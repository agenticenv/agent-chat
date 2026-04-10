// Package stream provides the pub/sub broker, bridge runner, and wire event
// types used for SSE streaming of agent runs to HTTP clients.
package stream

import (
	"errors"
	"time"

	"github.com/agenticenv/agent-chat/server/store"
)

// ErrTopicExists is returned by Broker.Open when a run is already in progress
// for a given conversation ID.
var ErrTopicExists = errors.New("stream: run already in progress for this conversation")

// EventType identifies the kind of SSE event sent to the browser.
type EventType string

const (
	// EventToken is a partial content chunk (content_delta from the LLM).
	EventToken EventType = "token"
	// EventToolCall is emitted when the agent invokes a tool.
	EventToolCall EventType = "tool_call"
	// EventToolResult is emitted when a tool execution completes.
	EventToolResult EventType = "tool_result"
	// EventError signals a terminal agent error.
	EventError EventType = "error"
	// EventDone is the terminal success event; Message carries the final
	// assistant message as persisted in the database.
	EventDone EventType = "done"
)

// Event is the JSON shape written to the browser as an SSE data frame.
// Only fields relevant to the EventType are populated.
type Event struct {
	Type       EventType     `json:"type"`
	Content    string        `json:"content,omitempty"`
	ToolName   string        `json:"tool_name,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	Result     interface{}   `json:"result,omitempty"`
	Message    *store.Message `json:"message,omitempty"` // set on EventDone
	Timestamp  time.Time     `json:"timestamp"`
}
