package stream

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/agenticenv/agent-chat/server/store"
	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
)

// Runner launches and manages bridge goroutines that translate SDK AgentEvents into wire Events and publish them to the Broker.
// Each bridge goroutine runs on a context derived from parentCtx (the server-level context), NOT from any HTTP request context.
// This ensures that a client disconnect does not cancel the underlying Temporal agent workflow.
type Runner struct {
	agent     *sdkagent.Agent
	broker    *Broker
	messages  *store.MessageStore
	rootAgent string          // name of the main agent; used to filter sub-agent Complete events
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

		switch evType := ev.Type(); evType {

		case sdkagent.AgentEventTypeTextMessageContent:
			if t, ok := ev.(*sdkagent.AgentTextMessageContentEvent); ok && t.Delta != "" {
				r.broker.Publish(convID, Event{
					Type:      EventToken,
					Content:   t.Delta,
					Timestamp: agentEventTime(ev),
				})
			}

		case sdkagent.AgentEventTypeToolCallStart:
			if t, ok := ev.(*sdkagent.AgentToolCallStartEvent); ok {
				r.broker.Publish(convID, Event{
					Type:       EventToolCall,
					ToolName:   t.ToolCallName,
					ToolCallID: t.ToolCallID,
					Timestamp:  agentEventTime(ev),
				})
			}

		case sdkagent.AgentEventTypeToolCallResult:
			if t, ok := ev.(*sdkagent.AgentToolCallResultEvent); ok {
				r.broker.Publish(convID, Event{
					Type: EventToolResult,
					//ToolName:  t.ToolCallName,
					Result:    t.Content,
					Timestamp: agentEventTime(ev),
				})
			}

		case sdkagent.AgentEventTypeRunError:
			if re, ok := ev.(*sdkagent.AgentRunErrorEvent); ok {
				r.broker.Publish(convID, Event{
					Type:      EventError,
					Content:   re.Message,
					Timestamp: agentEventTime(ev),
				})
			}
			return // defer closes topic

		case sdkagent.AgentEventTypeRunFinished:
			fin, res := runResultFromFinishedEvent(ev)
			if fin == nil {
				return
			}
			if res == nil {
				res = parseRunResultFromFinished(fin)
			}
			// Sub-agent RUN_FINISHED is ignored; root completion always emits done
			// (even when streaming left AgentRunResult.Content empty).
			if res != nil && res.AgentName != "" && res.AgentName != r.rootAgent {
				slog.Debug("stream: ignoring sub-agent complete", "agent", res.AgentName, "conv", convID)
				continue
			}
			done := r.buildDoneEvent(ctx, convID, agentEventTime(fin), runStartTime)
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

func runResultFromFinishedEvent(ev sdkagent.AgentEvent) (*sdkagent.AgentRunFinishedEvent, *sdkagent.AgentRunResult) {
	if ev == nil || ev.Type() != sdkagent.AgentEventTypeRunFinished {
		return nil, nil
	}
	fin, ok := ev.(*sdkagent.AgentRunFinishedEvent)
	if !ok || fin == nil {
		return nil, nil
	}
	res, _ := fin.Result.(*sdkagent.AgentRunResult)
	return fin, res
}

// parseRunResultFromFinished recovers [*sdkagent.AgentRunResult] after events round-trip through JSON
// (e.g. event bus decode leaves Result as map[string]any, so a plain type assertion fails).
func parseRunResultFromFinished(fin *sdkagent.AgentRunFinishedEvent) *sdkagent.AgentRunResult {
	if fin == nil || fin.Result == nil {
		return nil
	}
	b, err := json.Marshal(fin.Result)
	if err != nil {
		return nil
	}
	var out sdkagent.AgentRunResult
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return &out
}

// agentEventTime interprets BaseEvent timestamps as Unix milliseconds (AG-UI / SDK NewBaseEvent).
func agentEventTime(ev sdkagent.AgentEvent) time.Time {
	if ev == nil {
		return time.Now()
	}
	ts := ev.Timestamp()
	if ts == nil {
		return time.Now()
	}
	return time.UnixMilli(*ts)
}
