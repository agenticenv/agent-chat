// Package agent provides the session workflow that wraps the SDK's AgentWorkflow
// in a long-lived, per-conversation Temporal workflow.
package agent

import (
	sdkagent "github.com/vvsynapse/temporal-agent-sdk-go/pkg/agent"
	"go.temporal.io/sdk/workflow"
)

// SessionInput is the input to SessionWorkflow.
type SessionInput struct {
	ConversationID string `json:"conversation_id"`
	TaskQueue      string `json:"task_queue"`
}

// SessionWorkflow is a long-running workflow that stays alive for the duration
// of a conversation. Each user message arrives via a "send-message" update,
// which delegates to the SDK's AgentWorkflow as a child workflow.
//
// Workflow ID convention: "agent-session-{conversationID}"
func SessionWorkflow(ctx workflow.Context, input SessionInput) error {
	// Register update handler — each "send-message" update runs a child AgentWorkflow
	err := workflow.SetUpdateHandler(ctx, "send-message",
		func(ctx workflow.Context, content string) (*sdkagent.AgentResponse, error) {
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				TaskQueue: input.TaskQueue,
			})
			wfInput := sdkagent.AgentWorkflowInput{
				UserPrompt:     content,
				ConversationID: input.ConversationID,
			}
			var resp sdkagent.AgentResponse
			if err := workflow.ExecuteChildWorkflow(childCtx, "AgentWorkflow", wfInput).Get(ctx, &resp); err != nil {
				return nil, err
			}
			return &resp, nil
		},
	)
	if err != nil {
		return err
	}

	// Stay alive until "complete" signal (conversation deleted or server shutdown)
	completeCh := workflow.GetSignalChannel(ctx, "complete")
	completeCh.Receive(ctx, nil)
	return nil
}
