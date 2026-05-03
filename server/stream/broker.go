package stream

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

const subChannelBuf = 64

// Subscription is a handle returned by Broker.Subscribe. The caller reads
// raw JSON payloads from Ch; the channel is closed when the topic ends.
type Subscription struct {
	Ch      <-chan []byte
	ch      chan []byte
	topicID string
}

// topic represents one active agent run (one per conversation).
type topic struct {
	subs   map[*Subscription]struct{}
	cancel context.CancelFunc // cancels the bridge goroutine; stored for CloseAll
}

// Broker is an in-memory pub/sub hub. Each active agent run owns one topic
// keyed by conversation ID. It is safe for concurrent use.
type Broker struct {
	mu     sync.Mutex
	topics map[string]*topic
}

// NewBroker creates an empty Broker.
func NewBroker() *Broker {
	return &Broker{topics: make(map[string]*topic)}
}

// Open registers a new topic for topicID and stores cancel so CloseAll can
// cancel the associated bridge goroutine. Returns ErrTopicExists if a run is
// already active for that conversation.
func (b *Broker) Open(topicID string, cancel context.CancelFunc) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.topics[topicID]; exists {
		return ErrTopicExists
	}
	b.topics[topicID] = &topic{
		subs:   make(map[*Subscription]struct{}),
		cancel: cancel,
	}
	return nil
}

// Subscribe adds a new subscriber to topicID. Returns (nil, false) if the
// topic does not exist (e.g. the run completed between Start and Subscribe).
func (b *Broker) Subscribe(topicID string) (*Subscription, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.topics[topicID]
	if !ok {
		return nil, false
	}
	ch := make(chan []byte, subChannelBuf)
	sub := &Subscription{Ch: ch, ch: ch, topicID: topicID}
	t.subs[sub] = struct{}{}
	return sub, true
}

// Unsubscribe removes sub from its topic. Safe to call multiple times.
func (b *Broker) Unsubscribe(sub *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.topics[sub.topicID]
	if !ok {
		return
	}
	if _, member := t.subs[sub]; member {
		delete(t.subs, sub)
		close(sub.ch)
	}
}

// Publish delivers payload (AG-UI or app extension JSON) to every subscriber
// of topicID. The send is non-blocking per subscriber: if a buffer is full,
// the payload is dropped for that subscriber and a warning is logged.
func (b *Broker) Publish(topicID string, payload []byte) {
	b.mu.Lock()
	t, ok := b.topics[topicID]
	var subs []*Subscription
	if ok {
		subs = make([]*Subscription, 0, len(t.subs))
		for s := range t.subs {
			subs = append(subs, s)
		}
	}
	b.mu.Unlock()

	tname := agUIEventTypeName(payload)
	for _, s := range subs {
		select {
		case s.ch <- payload:
		default:
			slog.Warn("stream: subscriber buffer full, dropping event",
				"topic", topicID, "event_type", tname)
		}
	}
}

func agUIEventTypeName(payload []byte) string {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &head); err != nil || head.Type == "" {
		return "(unknown)"
	}
	return head.Type
}

// Close marks a topic done, closes every subscriber's channel, and removes
// the topic. Idempotent.
func (b *Broker) Close(topicID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.topics[topicID]
	if !ok {
		return
	}
	for s := range t.subs {
		close(s.ch)
	}
	delete(b.topics, topicID)
}

// CloseAll cancels every in-flight bridge context and closes all topics.
// Called during graceful server shutdown.
func (b *Broker) CloseAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, t := range b.topics {
		if t.cancel != nil {
			t.cancel()
		}
		for s := range t.subs {
			close(s.ch)
		}
		delete(b.topics, id)
	}
}

// Exists reports whether a run is currently active for topicID.
func (b *Broker) Exists(topicID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.topics[topicID]
	return ok
}
