package stream

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/agenticenv/agent-chat/server/store"
	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
)

// Runner launches and manages bridge goroutines that forward SDK AgentEvents
// (AG-UI JSON via ToJSON) to the Broker. After RUN_FINISHED, emits
// MESSAGE_PERSISTED with the DB row when resolved.
// Each bridge goroutine runs on a context derived from parentCtx (the server-level context), NOT from any HTTP request context.
// This ensures that a client disconnect does not cancel the underlying Temporal agent workflow.
type Runner struct {
	agent     *sdkagent.Agent
	broker    *Broker
	messages  *store.MessageStore
	parentCtx context.Context // server-level; canceled only on graceful shutdown
}

// NewRunner creates a Runner.
func NewRunner(
	a *sdkagent.Agent,
	b *Broker,
	ms *store.MessageStore,
	parent context.Context,
) *Runner {
	return &Runner{
		agent:     a,
		broker:    b,
		messages:  ms,
		parentCtx: parent,
	}
}

// Start opens a broker topic for convID and launches the bridge goroutine.
// Returns ErrTopicExists if a run is already in progress for this conversation.
func (r *Runner) Start(convID, content string) error {
	runCtx, cancel := context.WithCancel(r.parentCtx)

	if err := r.broker.Open(convID, cancel); err != nil {
		cancel()
		return err
	}

	go r.run(runCtx, convID, content)
	return nil
}

// run publishes ev.ToJSON() for every stream event. RUN_FINISHED is followed by MESSAGE_PERSISTED (DB attach).
func (r *Runner) run(ctx context.Context, convID, content string) {
	defer r.broker.Close(convID)

	runStartTime := time.Now()

	eventCh, err := r.agent.Stream(ctx, content, convID)
	if err != nil {
		slog.Error("stream: agent.Stream failed", "conv", convID, "err", err)
		payload, mErr := wireRunErrorJSON(err.Error())
		if mErr != nil {
			slog.Error("stream: marshal RUN_ERROR", "err", mErr)
			return
		}
		r.broker.Publish(convID, payload)
		return
	}

	publish := func(payload []byte) {
		if len(payload) == 0 {
			return
		}
		r.broker.Publish(convID, payload)
	}

	for ev := range eventCh {
		if ev == nil {
			continue
		}
		if t, ok := ev.(*sdkagent.AgentTextMessageContentEvent); ok && t.Delta == "" {
			continue
		}

		payload, err := ev.ToJSON()
		if err != nil {
			slog.Debug("stream: ToJSON skip", "conv", convID, "type", ev.Type(), "err", err)
			continue
		}
		publish(payload)

		if ev.Type() == sdkagent.AgentEventTypeRunFinished {
			if mp, err := r.buildMessagePersistedJSON(ctx, convID, runStartTime); err != nil {
				slog.Warn("stream: MESSAGE_PERSISTED marshal", "conv", convID, "err", err)
			} else {
				publish(mp)
			}
			return
		}
	}

	mp, err := r.buildMessagePersistedJSON(ctx, convID, runStartTime)
	if err != nil {
		slog.Warn("stream: MESSAGE_PERSISTED (channel closed)", "conv", convID, "err", err)
		return
	}
	publish(mp)
}

// buildMessagePersistedJSON returns JSON for type MESSAGE_PERSISTED, resolving the assistant row from the DB.
func (r *Runner) buildMessagePersistedJSON(ctx context.Context, convID string, runStartTime time.Time) ([]byte, error) {
	ev := MessagePersistedWire{Type: WireTypeMessagePersisted, Timestamp: time.Now()}
	lookAfter := runStartTime.Add(-2 * time.Second)

	for i := 0; i < 10; i++ {
		msgs, err := r.messages.List(ctx, convID)
		if err == nil {
			for j := len(msgs) - 1; j >= 0; j-- {
				if msgs[j].Role == "assistant" && msgs[j].CreatedAt.After(lookAfter) {
					m := msgs[j]
					ev.Message = &m
					return json.Marshal(ev)
				}
			}
		}
		if i < 9 {
			select {
			case <-ctx.Done():
				slog.Warn("stream: new assistant message not found (ctx done)", "conv", convID)
				return json.Marshal(ev)
			case <-time.After(150 * time.Millisecond):
			}
		}
	}

	slog.Warn("stream: new assistant message not found in DB after retries", "conv", convID)
	return json.Marshal(ev)
}

func wireRunErrorJSON(message string) ([]byte, error) {
	ts := time.Now().UnixMilli()
	t := ts
	return json.Marshal(struct {
		Type      string `json:"type"`
		Timestamp *int64 `json:"timestamp,omitempty"`
		Message   string `json:"message"`
	}{
		Type:      string(sdkagent.AgentEventTypeRunError),
		Timestamp: &t,
		Message:   message,
	})
}
