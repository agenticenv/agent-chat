// Package agent provides the bridge between our PostgreSQL store and the
// agent-sdk-go interfaces.Conversation contract.
package agent

import (
	"context"

	"github.com/agenticenv/agent-chat/server/store"
	"github.com/agenticenv/agent-sdk-go/pkg/interfaces"
)

// PGConversation implements interfaces.Conversation backed by PostgreSQL.
// It is single-process (IsDistributed returns false) so it works with the
// embedded Temporal worker that runs in the same process.
//
// Persistence strategy:
//   - The handler always writes the user message before calling agent.Run.
//   - The SDK workflow calls AddMessage for the assistant reply after the LLM responds.
//   - ListMessages returns everything from Postgres, so both the REST API and
//     the SDK's LLM context window share the same source of truth.
type PGConversation struct {
	store *store.MessageStore
}

// NewPGConversation creates a PGConversation wrapping the given MessageStore.
func NewPGConversation(s *store.MessageStore) *PGConversation {
	return &PGConversation{store: s}
}

// AddMessage persists a message to the conversation. Called by the Temporal
// workflow via AddConversationMessagesActivity after each run completes.
// The SDK owns all persistence — user, assistant, and tool messages are all
// written here. The handler no longer pre-writes messages before agent.Run.
func (c *PGConversation) AddMessage(ctx context.Context, conversationID string, msg interfaces.Message) error {
	_, err := c.store.Create(ctx, conversationID, string(msg.Role), msg.Content)
	return err
}

// ListMessages returns stored messages for the given conversation, converting
// from the store type to the interfaces.Message type expected by the SDK.
// The opts (Limit, Offset, Roles filters) are applied in-memory after fetching
// from Postgres — sufficient for typical conversation window sizes.
func (c *PGConversation) ListMessages(ctx context.Context, conversationID string, opts ...interfaces.ListMessagesOption) ([]interfaces.Message, error) {
	stored, err := c.store.List(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Parse options.
	o := &interfaces.ListMessagesOptions{}
	for _, opt := range opts {
		opt(o)
	}

	msgs := make([]interfaces.Message, 0, len(stored))
	for _, m := range stored {
		role := interfaces.MessageRole(m.Role)

		// Apply role filter if specified.
		if len(o.Roles) > 0 && !containsRole(o.Roles, role) {
			continue
		}

		msgs = append(msgs, interfaces.Message{
			Role:      role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		})
	}

	// Apply offset.
	if o.Offset > 0 {
		if o.Offset >= len(msgs) {
			return []interfaces.Message{}, nil
		}
		msgs = msgs[o.Offset:]
	}

	// Apply limit.
	if o.Limit > 0 && len(msgs) > o.Limit {
		msgs = msgs[:o.Limit]
	}

	return msgs, nil
}

// Clear is a no-op: conversations are cleared only when the user explicitly
// deletes them via the REST API (which cascades to the messages table).
func (c *PGConversation) Clear(_ context.Context, _ string) error {
	return nil
}

// IsDistributed returns false — the agent and its embedded Temporal worker run
// in the same process, so distributed storage is not required.
func (c *PGConversation) IsDistributed() bool {
	return false
}

// Ensure PGConversation satisfies the interface at compile time.
var _ interfaces.Conversation = (*PGConversation)(nil)

// containsRole checks whether role is in the slice.
func containsRole(roles []interfaces.MessageRole, role interfaces.MessageRole) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
