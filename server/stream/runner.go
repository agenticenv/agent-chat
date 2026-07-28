package stream

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/agenticenv/agent-chat/server/store"
	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
)

// Runner launches and manages bridge goroutines that forward SDK AgentEvents
// (AG-UI JSON via ToJSON) to the Broker. After RUN_FINISHED, emits
// MESSAGE_PERSISTED with the DB row when resolved.
//
// agent.Stream / GetAgentStream use a non-cancellable context so server shutdown
// and client disconnect never cancel the Temporal workflow. Only the Events
// subscription context is cancelled (via Broker.CloseAll) to detach locally.
type Runner struct {
	agent     *sdkagent.Agent
	broker    *Broker
	messages  *store.MessageStore
	convs     *store.ConversationStore
	parentCtx context.Context // process-lifetime; not cancelled on HTTP disconnect
}

// NewRunner creates a Runner.
func NewRunner(
	a *sdkagent.Agent,
	b *Broker,
	ms *store.MessageStore,
	cs *store.ConversationStore,
	parent context.Context,
) *Runner {
	return &Runner{
		agent:     a,
		broker:    b,
		messages:  ms,
		convs:     cs,
		parentCtx: parent,
	}
}

// Start opens a broker topic for convID and launches a new agent.Stream bridge.
// Returns ErrTopicExists if a run is already in progress for this conversation.
func (r *Runner) Start(convID, content string) error {
	eventsCtx, cancelEvents := context.WithCancel(r.parentCtx)

	if err := r.broker.Open(convID, cancelEvents); err != nil {
		cancelEvents()
		return err
	}

	go r.streamNew(eventsCtx, convID, content)
	return nil
}

// Resume opens a broker topic and bridges GetAgentStream events from offset.
// Returns ErrTopicExists if a live bridge already owns this conversation
// (caller should Subscribe to the existing topic instead).
func (r *Runner) Resume(convID, streamID string, offset int64) error {
	eventsCtx, cancelEvents := context.WithCancel(r.parentCtx)

	if err := r.broker.Open(convID, cancelEvents); err != nil {
		cancelEvents()
		return err
	}

	go r.streamResume(eventsCtx, convID, streamID, offset)
	return nil
}

// streamCtx is used for agent.Stream / GetAgentStream / DB writes so cancelling
// the Events subscriber never cancels the Temporal agent run.
func (r *Runner) streamCtx() context.Context {
	return context.WithoutCancel(r.parentCtx)
}

func (r *Runner) streamNew(eventsCtx context.Context, convID, content string) {
	defer r.broker.Close(convID)

	runStartTime := time.Now()
	progressStarted := false
	streamCtx := r.streamCtx()

	streamOpts := sdkagent.AgentStreamOptions{
		ConversationOptions: &sdkagent.ConversationOptions{
			ID: convID,
		},
	}
	agentStream, err := r.agent.Stream(streamCtx, content, &streamOpts)
	if err != nil {
		slog.Error("stream: agent.Stream failed", "conv", convID, "err", err)
		r.publishRunError(convID, err.Error())
		return
	}

	events, err := agentStream.Events(eventsCtx)
	if err != nil {
		slog.Error("stream: agentStream.Events failed", "conv", convID, "err", err)
		return
	}

	r.forwardEvents(streamCtx, convID, agentStream.ID(), events, &progressStarted, runStartTime)
}

func (r *Runner) streamResume(eventsCtx context.Context, convID, streamID string, offset int64) {
	defer r.broker.Close(convID)

	runStartTime := time.Now()
	progressStarted := true // conversation row already status=running
	streamCtx := r.streamCtx()

	slog.Info("stream: resume bridge start", "conv", convID, "stream", streamID, "offset", offset)

	agentStream, err := r.agent.GetAgentStream(streamCtx, streamID)
	if err != nil {
		if errors.Is(err, sdkagent.ErrRunAlreadyCompleted) {
			slog.Info("stream: resume target already completed", "conv", convID, "stream", streamID)
			if mErr := r.convs.MarkCompleted(streamCtx, convID); mErr != nil {
				slog.Warn("stream: mark completed after already-done resume", "conv", convID, "err", mErr)
			}
			return
		}
		slog.Error("stream: GetAgentStream failed", "conv", convID, "stream", streamID, "err", err)
		r.publishRunError(convID, err.Error())
		return
	}

	slog.Info("stream: GetAgentStream ok", "conv", convID, "stream", streamID)

	var events <-chan sdkagent.AgentEvent
	if offset > 0 {
		events, err = agentStream.Events(eventsCtx, sdkagent.WithOffset(offset))
	} else {
		events, err = agentStream.Events(eventsCtx)
	}
	if err != nil {
		if errors.Is(err, sdkagent.ErrRunAlreadyCompleted) {
			slog.Info("stream: resume Events already completed", "conv", convID, "stream", streamID)
			if mErr := r.convs.MarkCompleted(streamCtx, convID); mErr != nil {
				slog.Warn("stream: mark completed after already-done Events", "conv", convID, "err", mErr)
			}
			return
		}
		slog.Error("stream: resume Events failed", "conv", convID, "stream", streamID, "err", err)
		r.publishRunError(convID, err.Error())
		return
	}

	slog.Info("stream: resume Events subscribed", "conv", convID, "stream", streamID, "offset", offset)
	r.forwardEvents(streamCtx, convID, streamID, events, &progressStarted, runStartTime)
}

func (r *Runner) forwardEvents(
	ctx context.Context,
	convID, streamID string,
	events <-chan sdkagent.AgentEvent,
	progressStarted *bool,
	runStartTime time.Time,
) {
	loggedFirst := false
	for ev := range events {
		if ev == nil {
			continue
		}
		if t, ok := ev.(*sdkagent.AgentTextMessageContentEvent); ok && t.Delta == "" {
			continue
		}

		if !loggedFirst {
			off, hasOff := eventOffset(ev)
			slog.Info("stream: first event",
				"conv", convID,
				"stream", streamID,
				"type", string(ev.Type()),
				"offset", off,
				"has_offset", hasOff,
			)
			loggedFirst = true
		}

		r.trackProgress(ctx, convID, streamID, ev, progressStarted)

		payload, err := ev.ToJSON()
		if err != nil {
			slog.Debug("stream: ToJSON skip", "conv", convID, "type", ev.Type(), "err", err)
			continue
		}
		r.broker.Publish(convID, payload)

		switch ev.Type() {
		case sdkagent.AgentEventTypeRunFinished:
			slog.Info("stream: RUN_FINISHED", "conv", convID, "stream", streamID)
			if err := r.convs.MarkCompleted(ctx, convID); err != nil {
				slog.Warn("stream: mark completed", "conv", convID, "err", err)
			}
			if mp, err := r.buildMessagePersistedJSON(ctx, convID, runStartTime); err != nil {
				slog.Warn("stream: MESSAGE_PERSISTED marshal", "conv", convID, "err", err)
			} else {
				r.broker.Publish(convID, mp)
			}
			return
		case sdkagent.AgentEventTypeRunError:
			slog.Info("stream: RUN_ERROR", "conv", convID, "stream", streamID)
			if err := r.convs.MarkFailed(ctx, convID); err != nil {
				slog.Warn("stream: mark failed", "conv", convID, "err", err)
			}
			return
		}
	}

	slog.Info("stream: events channel closed", "conv", convID, "stream", streamID, "saw_events", loggedFirst)
	mp, err := r.buildMessagePersistedJSON(ctx, convID, runStartTime)
	if err != nil {
		slog.Warn("stream: MESSAGE_PERSISTED (channel closed)", "conv", convID, "err", err)
		return
	}
	r.broker.Publish(convID, mp)
}

func (r *Runner) publishRunError(convID, message string) {
	payload, mErr := wireRunErrorJSON(message)
	if mErr != nil {
		slog.Error("stream: marshal RUN_ERROR", "err", mErr)
		return
	}
	r.broker.Publish(convID, payload)
}

// trackProgress updates the conversation row from stream events.
func (r *Runner) trackProgress(ctx context.Context, convID, streamID string, ev sdkagent.AgentEvent, progressStarted *bool) {
	runID := streamID
	if started, ok := ev.(*sdkagent.AgentRunStartedEvent); ok && started.RunID != "" {
		runID = started.RunID
	} else if finished, ok := ev.(*sdkagent.AgentRunFinishedEvent); ok && finished.RunID != "" {
		runID = finished.RunID
	}

	offset, hasOffset := eventOffset(ev)

	if !*progressStarted {
		off := int64(0)
		if hasOffset {
			off = offset
		}
		if err := r.convs.SetRunning(ctx, convID, runID, off); err != nil {
			slog.Warn("stream: set running", "conv", convID, "err", err)
			return
		}
		*progressStarted = true
		return
	}

	if hasOffset {
		if err := r.convs.UpdateOffset(ctx, convID, offset); err != nil {
			slog.Warn("stream: update offset", "conv", convID, "err", err)
		}
	}
}

type offsetCarrier interface {
	Offset() (int64, bool)
}

func eventOffset(ev sdkagent.AgentEvent) (int64, bool) {
	if o, ok := ev.(offsetCarrier); ok {
		return o.Offset()
	}
	return 0, false
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
