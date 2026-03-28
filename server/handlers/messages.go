package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vvsynapse/agent-demo/server/store"
	sdkagent "github.com/vvsynapse/agent-sdk-go/pkg/agent"
)

type MessageHandler struct {
	store     *store.MessageStore
	convStore *store.ConversationStore
	agent     *sdkagent.Agent
}

func NewMessageHandler(ms *store.MessageStore, cs *store.ConversationStore, a *sdkagent.Agent) *MessageHandler {
	return &MessageHandler{store: ms, convStore: cs, agent: a}
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
