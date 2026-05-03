// Package stream provides the pub/sub broker, bridge runner, and wire types
// used for SSE streaming of agent runs to HTTP clients.
package stream

import (
	"errors"
	"time"

	"github.com/agenticenv/agent-chat/server/store"
)

// ErrTopicExists is returned by Broker.Open when a run is already in progress
// for a given conversation ID.
var ErrTopicExists = errors.New("stream: run already in progress for this conversation")

// WireTypeMessagePersisted is an app-level SSE frame (not part of core AG-UI)
// emitted after [RUN_FINISHED] so clients can attach the persisted DB message id.
const WireTypeMessagePersisted = "MESSAGE_PERSISTED"

// MessagePersistedWire is JSON sent after the AG-UI RUN_FINISHED event when
// the assistant row has been resolved from the message store.
type MessagePersistedWire struct {
	Type      string         `json:"type"`
	Message   *store.Message `json:"message,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}
