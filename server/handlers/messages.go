package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/agenticenv/agent-chat/server/store"
	"github.com/agenticenv/agent-chat/server/stream"
)

type MessageHandler struct {
	store     *store.MessageStore
	convStore *store.ConversationStore
	runner    *stream.Runner
	broker    *stream.Broker
}

func NewMessageHandler(
	ms *store.MessageStore,
	cs *store.ConversationStore,
	runner *stream.Runner,
	broker *stream.Broker,
) *MessageHandler {
	return &MessageHandler{
		store:     ms,
		convStore: cs,
		runner:    runner,
		broker:    broker,
	}
}

// GET /api/conversations/{id}/messages
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "id")
	msgs, err := h.store.List(r.Context(), convID)
	if err != nil {
		jsonError(w, "failed to list messages", http.StatusInternalServerError)
		return
	}
	jsonOK(w, msgs)
}

// POST /api/conversations/{id}/messages
//
// Starts the agent run in a background goroutine (decoupled from this HTTP
// request's context) and streams AG-UI JSON events (SDK ToJSON) as SSE frames,
// followed by an app extension frame type MESSAGE_PERSISTED after root RUN_FINISHED.
// A client disconnect does NOT cancel the agent run — it continues in the
// background and the final state is retrievable via GET /messages.
func (h *MessageHandler) Stream(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "id")

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Content == "" {
		jsonError(w, "content is required", http.StatusBadRequest)
		return
	}

	exists, err := h.convStore.Exists(r.Context(), convID)
	if err != nil {
		jsonError(w, "failed to check conversation", http.StatusInternalServerError)
		return
	}
	if !exists {
		jsonError(w, "conversation not found", http.StatusNotFound)
		return
	}

	if err := h.runner.Start(convID, body.Content); err != nil {
		if errors.Is(err, stream.ErrTopicExists) {
			jsonError(w, "a run is already in progress for this conversation", http.StatusConflict)
			return
		}
		jsonError(w, "failed to start agent run: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.serveSSE(w, r, convID)
}

// POST /api/conversations/{id}/resume
//
// If the conversation status is running, opens an SSE stream of remaining agent
// events (GetAgentStream + offset from the conversation row). If a live bridge
// already exists for this conversation, reattaches to it. No request body.
func (h *MessageHandler) Resume(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "id")

	conv, err := h.convStore.Get(r.Context(), convID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			jsonError(w, "conversation not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to load conversation", http.StatusInternalServerError)
		return
	}

	slog.Info("resume: request",
		"conv", convID,
		"status", conv.Status,
		"stream", conv.AgentStreamID,
		"offset", conv.LastEventOffset,
	)

	if conv.Status != store.ConversationStatusRunning {
		slog.Info("resume: reject not running", "conv", convID, "status", conv.Status)
		jsonError(w, "conversation is not running", http.StatusConflict)
		return
	}
	if conv.AgentStreamID == "" {
		slog.Info("resume: reject missing stream id", "conv", convID)
		jsonError(w, "conversation has no agent stream id", http.StatusConflict)
		return
	}

	// Live bridge still publishing (e.g. client disconnected earlier): just subscribe.
	if h.broker.Exists(convID) {
		slog.Info("resume: reattach existing broker topic", "conv", convID)
	} else {
		slog.Info("resume: start GetAgentStream bridge",
			"conv", convID,
			"stream", conv.AgentStreamID,
			"offset", conv.LastEventOffset,
		)
		if err := h.runner.Resume(convID, conv.AgentStreamID, conv.LastEventOffset); err != nil {
			if errors.Is(err, stream.ErrTopicExists) {
				jsonError(w, "a run is already in progress for this conversation", http.StatusConflict)
				return
			}
			jsonError(w, "failed to resume agent stream: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	h.serveSSE(w, r, convID)
}

// serveSSE subscribes to the broker topic and writes SSE frames until the topic
// closes or the client disconnects. The agent bridge is independent of this HTTP request.
func (h *MessageHandler) serveSSE(w http.ResponseWriter, r *http.Request, convID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported by server", http.StatusInternalServerError)
		return
	}

	sub, ok := h.broker.Subscribe(convID)
	if !ok {
		// Rare race: topic closed between Start/Resume and Subscribe.
		slog.Info("sse: topic gone before subscribe", "conv", convID)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	defer h.broker.Unsubscribe(sub)

	slog.Info("sse: subscribed", "conv", convID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	clientGone := r.Context().Done()
	frames := 0
	for {
		select {
		case payload, open := <-sub.Ch:
			if !open {
				slog.Info("sse: topic closed", "conv", convID, "frames", frames)
				return
			}
			frames++
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				slog.Info("sse: write failed (client gone?)", "conv", convID, "frames", frames, "err", err)
				return
			}
			flusher.Flush()

		case <-clientGone:
			slog.Info("sse: client disconnected", "conv", convID, "frames", frames)
			return
		}
	}
}
