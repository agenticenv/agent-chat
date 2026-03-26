package handlers

import (
	"encoding/json"
	"net/http"
	"github.com/go-chi/chi/v5"
	sdkagent "github.com/vvsynapse/temporal-agent-sdk-go/pkg/agent"
	"github.com/vvsynapse/agent-demo/server/store"
)

type MessageHandler struct {
	store *store.MessageStore
	convStore *store.ConversationStore
	agent *sdkagent.Agent
}

func NewMessageHandler(store *store.MessageStore, convStore *store.ConversationStore, agent *sdkagent.Agent) *MessageHandler {
	return &MessageHandler{
		store: store,
		convStore: convStore,
		agent: agent,
	}
}

//GET /api/conversations/{conversationID}/messages
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "id")
	msgs, err := h.store.List(r.Context(), convID)
	if err != nil {
		jsonError(w, "failed to list messages", http.StatusInternalServerError)
		return
	}
	jsonOK(w, msgs)
}

//POST /api/conversations/{conversationID}/messages
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "id")
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Content == "" {
		jsonError(w, "content is required", http.StatusBadRequest)
		return
	}
	// Validate conversation exists
	exists, err := h.convStore.Exists(r.Context(), convID)
	if err != nil {
		jsonError(w, "failed to check conversation", http.StatusInternalServerError)
		return
	}
	if !exists {
		jsonError(w, "conversation not found", http.StatusNotFound)
		return
	}

	// SDK owns persistence: agent.Run triggers the Temporal workflow which calls
	// AddConversationMessagesActivity → PGConversation.AddMessage for all messages
	// (user + assistant). No pre/post writes here to avoid duplicates.
	if _, err := h.agent.Run(r.Context(), body.Content, convID); err != nil {
		jsonError(w, "agent error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch all messages and return the last one (the assistant reply) so the UI gets a real DB-generated ID.
	msgs, err := h.store.List(r.Context(), convID)
	if err != nil || len(msgs) == 0 {
		jsonError(w, "failed to retrieve reply", http.StatusInternalServerError)
		return
	}
	jsonOK(w, msgs[len(msgs)-1])
}