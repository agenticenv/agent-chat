package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/agenticenv/agent-chat/server/store"
	"github.com/go-chi/chi/v5"
)

// ConversationHandler handles CRUD for conversations.
type ConversationHandler struct {
	store *store.ConversationStore
}

func NewConversationHandler(s *store.ConversationStore) *ConversationHandler {
	return &ConversationHandler{store: s}
}

// GET /api/conversations
func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	convs, err := h.store.List(r.Context())
	if err != nil {
		jsonError(w, "failed to list conversations", http.StatusInternalServerError)
		return
	}
	jsonOK(w, convs)
}

// POST /api/conversations  body: {"title":"..."}
func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		body.Title = "New chat"
	}
	conv, err := h.store.Create(r.Context(), body.Title)
	if err != nil {
		jsonError(w, "failed to create conversation", http.StatusInternalServerError)
		return
	}
	jsonOK(w, conv)
}

// PATCH /api/conversations/{id}  body: {"title":"..."}
func (h *ConversationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}
	if err := h.store.Update(r.Context(), id, body.Title); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			jsonError(w, "conversation not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to update conversation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/conversations/{id}
func (h *ConversationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			jsonError(w, "conversation not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to delete conversation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
