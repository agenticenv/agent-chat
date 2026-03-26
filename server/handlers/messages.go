package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	agentconv "github.com/vvsynapse/agent-demo/server/agent"
	"github.com/vvsynapse/agent-demo/server/store"
	sdkagent "github.com/vvsynapse/temporal-agent-sdk-go/pkg/agent"
)

type MessageHandler struct {
	store            *store.MessageStore
	convStore        *store.ConversationStore
	temporalClient   client.Client
	sessionTaskQueue string // task queue for SessionWorkflow
	agentTaskQueue   string // task queue for AgentWorkflow (SDK activities)
}

func NewMessageHandler(ms *store.MessageStore, cs *store.ConversationStore, tc client.Client, sessionTaskQueue, agentTaskQueue string) *MessageHandler {
	return &MessageHandler{
		store:            ms,
		convStore:        cs,
		temporalClient:   tc,
		sessionTaskQueue: sessionTaskQueue,
		agentTaskQueue:   agentTaskQueue,
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

	// UpdateWithStartWorkflow: starts the session workflow if not running,
	// or sends the update to the existing one. The session workflow delegates
	// to the SDK's AgentWorkflow as a child workflow.
	workflowID := "agent-session-" + convID

	startOp := h.temporalClient.NewWithStartWorkflowOperation(
		client.StartWorkflowOptions{
			ID:                       workflowID,
			TaskQueue:                h.sessionTaskQueue,
			WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		},
		"SessionWorkflow",
		agentconv.SessionInput{ConversationID: convID, TaskQueue: h.agentTaskQueue},
	)

	handle, err := h.temporalClient.UpdateWithStartWorkflow(r.Context(), client.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startOp,
		UpdateOptions: client.UpdateWorkflowOptions{
			UpdateName:   "send-message",
			Args:         []interface{}{body.Content},
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	})
	if err != nil {
		jsonError(w, "agent error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var resp sdkagent.AgentResponse
	if err := handle.Get(r.Context(), &resp); err != nil {
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
