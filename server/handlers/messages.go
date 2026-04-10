package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/agenticenv/agent-chat/server/store"
	"github.com/agenticenv/agent-chat/server/stream"
	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
)

type MessageHandler struct {
	store     *store.MessageStore
	convStore *store.ConversationStore
	agent     *sdkagent.Agent
	runner    *stream.Runner
	broker    *stream.Broker
}

func NewMessageHandler(
	ms *store.MessageStore,
	cs *store.ConversationStore,
	a *sdkagent.Agent,
	runner *stream.Runner,
	broker *stream.Broker,
) *MessageHandler {
	return &MessageHandler{
		store:     ms,
		convStore: cs,
		agent:     a,
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
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
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

	// SDK handles everything: Temporal workflow, LLM call, message persistence.
	if _, err := h.agent.Run(r.Context(), body.Content, convID); err != nil {
		jsonError(w, "agent error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch last message from DB (the assistant reply with its DB-generated ID)
	msgs, err := h.store.List(r.Context(), convID)
	if err != nil || len(msgs) == 0 {
		jsonError(w, "failed to retrieve reply", http.StatusInternalServerError)
		return
	}
	jsonOK(w, msgs[len(msgs)-1])
}

// POST /api/conversations/{id}/messages/stream
//
// Starts the agent run in a background goroutine (decoupled from this HTTP
// request's context) and streams AgentEvents to the client as SSE frames.
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported by server", http.StatusInternalServerError)
		return
	}

	// Start the bridge goroutine. Returns ErrTopicExists if a run is already
	// active for this conversation (only one run per conversation at a time).
	if err := h.runner.Start(convID, body.Content); err != nil {
		if errors.Is(err, stream.ErrTopicExists) {
			jsonError(w, "a run is already in progress for this conversation", http.StatusConflict)
			return
		}
		jsonError(w, "failed to start agent run: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Subscribe to the broker topic the runner just opened.
	sub, ok := h.broker.Subscribe(convID)
	if !ok {
		// Rare race: topic closed between Start and Subscribe (e.g. instant error).
		// Return 204 so the client falls back to GET /messages.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	defer h.broker.Unsubscribe(sub)

	// Write SSE headers before any body bytes.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx/proxy buffering
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Drain loop: forward broker events to the client as SSE frames.
	clientGone := r.Context().Done()
	for {
		select {
		case ev, open := <-sub.Ch:
			if !open {
				// Broker closed the topic (run complete or server shutdown).
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				// Client write failed (disconnect). Return — the run keeps going.
				return
			}
			flusher.Flush()

			// After sending the terminal events we can close the connection cleanly.
			if ev.Type == stream.EventDone || ev.Type == stream.EventError {
				return
			}

		case <-clientGone:
			// Client disconnected. Unsubscribe via defer; runner is unaffected.
			return
		}
	}
}
