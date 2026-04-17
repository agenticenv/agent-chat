package stream

import (
	"context"
	"log/slog"
	"time"
	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
	"github.com/agenticenv/agent-chat/server/store"
)

// Runner launches and manages bridge goroutines that translate SDK AgentEvents into wire Events and publish them to the Broker. 
// Each bridge goroutine runs on a context derived from parentCtx (the server-level context), NOT from any HTTP request context. 
// This ensures that a client disconnect does not cancel the underlying Temporal agent workflow.
type Runner struct {
	agent     *sdkagent.Agent
	broker    *Broker
	messages  *store.MessageStore
	rootAgent string         // name of the main agent; used to filter sub-agent Complete events
	parentCtx context.Context // server-level; canceled only on graceful shutdown
}

// NewRunner creates a Runner.
func NewRunner(
	a *sdkagent.Agent,
	b *Broker,
	ms *store.MessageStore,
	rootAgent string,
	parent context.Context,
) *Runner {
	return &Runner{
		agent:     a,
		broker:    b,
		messages:  ms,
		rootAgent: rootAgent,
		parentCtx: parent,
	}
}

// Start opens a broker topic for convID and launches the bridge goroutine.
// Returns ErrTopicExists if a run is already in progress for this conversation.
func (r *Runner) Start(convID, content string) error {
	runCtx, cancel := context.WithCancel(r.parentCtx)

	if err := r.broker.Open(convID, cancel); err != nil {
		cancel() // nothing to cancel yet, but clean up
		return err
	}

	go r.run(runCtx, convID, content)
	return nil
}

// run is the bridge goroutine. It owns runCtx and is the only place that
// touches the SDK stream for this conversation turn.
func (r *Runner) run(ctx context.Context, convID, content string) {
	// Always close the topic when we exit so all subscribers see channel close.
	defer r.broker.Close(convID)

	// Record the time before calling Stream so buildDoneEvent can distinguish the new assistant message from pre-existing ones in the DB. This is
	// necessary because AgentEventComplete fires before Temporal's AddConversationMessagesActivity finishes writing the message to Postgres.
	runStartTime := time.Now()

	eventCh, err := r.agent.Stream(ctx, content, convID)
	if err != nil {
		slog.Error("stream: agent.Stream failed", "conv", convID, "err", err)
		r.broker.Publish(convID, Event{
			Type:      EventError,
			Content:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	for ev := range eventCh {
		if ev == nil {
			continue
		}

		switch ev.Type {

		case sdkagent.AgentEventContentDelta:
			if ev.Content == "" {
				continue
			}
			r.broker.Publish(convID, Event{
				Type:      EventToken,
				Content:   ev.Content,
				Timestamp: ev.Timestamp,
			})

		case sdkagent.AgentEventToolCall:
			if ev.ToolCall == nil {
				continue
			}
			r.broker.Publish(convID, Event{
				Type:       EventToolCall,
				ToolName:   ev.ToolCall.ToolName,
				ToolCallID: ev.ToolCall.ToolCallID,
				Timestamp:  ev.Timestamp,
			})

		case sdkagent.AgentEventToolResult:
			if ev.ToolCall == nil {
				continue
			}
			r.broker.Publish(convID, Event{
				Type:      EventToolResult,
				ToolName:  ev.ToolCall.ToolName,
				Result:    ev.ToolCall.Result,
				Timestamp: ev.Timestamp,
			})

		case sdkagent.AgentEventError:
			errStr := "agent error"
			if ev.Error != nil {
				errStr = ev.Error.Error()
			}
			slog.Error("stream: agent error event", "conv", convID, "err", errStr)
			r.broker.Publish(convID, Event{
				Type:      EventError,
				Content:   errStr,
				Timestamp: ev.Timestamp,
			})
			return // defer closes topic

		case sdkagent.AgentEventComplete:
			// The SDK fans sub-agent Complete events into the same channel.
			// Only the ROOT agent's Complete is the terminal signal.
			// If AgentName is empty it means there are no sub-agents and the
			// event is from the root agent by definition.
			if ev.AgentName != "" && ev.AgentName != r.rootAgent {
				slog.Debug("stream: ignoring sub-agent complete", "agent", ev.AgentName, "conv", convID)
				continue
			}

			done := r.buildDoneEvent(ctx, convID, ev.Timestamp, runStartTime)
			r.broker.Publish(convID, done)
			return // defer closes topic

		// Explicitly skip events we don't surface in v1.
		// AgentEventContent duplicates the delta stream (README warns against printing both).
		// AgentEventThinking / AgentEventThinkingDelta: no UI yet.
		// AgentEventApproval: agent uses AutoToolApprovalPolicy, never fires.
		default:
			// skip
		}
	}

	// Channel closed without an explicit Complete (e.g. context canceled by
	// CloseAll on shutdown). Publish a best-effort done with whatever is in DB.
	done := r.buildDoneEvent(ctx, convID, time.Now(), runStartTime)
	r.broker.Publish(convID, done)
}

// buildDoneEvent waits for the assistant reply to appear in the DB and returns an EventDone carrying it.

// AgentEventComplete fires from the Temporal workflow before AddConversationMessagesActivity finishes writing the new message to Postgres.
// To bridge this race, we retry the DB read (up to 10 × 150 ms = 1.5 s) until we find an assistant message whose CreatedAt is after runStartTime. 
// A 2-second buffer is subtracted from runStartTime to absorb any clock skew between the
// Go server and Postgres.

// If the message never appears (e.g. the run was cancelled), Message is nil and
// the client falls back to fetching /messages on next load.

func (r *Runner) buildDoneEvent(ctx context.Context, convID string, ts time.Time, runStartTime time.Time) Event {
	ev := Event{Type: EventDone, Timestamp: ts}
	lookAfter := runStartTime.Add(-2 * time.Second)

	for i := 0; i < 10; i++ {
		msgs, err := r.messages.List(ctx, convID)
		if err == nil {
			for j := len(msgs) - 1; j >= 0; j-- {
				if msgs[j].Role == "assistant" && msgs[j].CreatedAt.After(lookAfter) {
					m := msgs[j]
					ev.Message = &m
					return ev
				}
			}
		}
		if i < 9 {
			select {
			case <-ctx.Done():
				return ev
			case <-time.After(150 * time.Millisecond):
			}
		}
	}

	slog.Warn("stream: new assistant message not found in DB after retries", "conv", convID)
	return ev
}
