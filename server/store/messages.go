package store

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId,omitempty"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

type MessageStore struct {
	pool *pgxpool.Pool
}

func NewMessageStore(pool *pgxpool.Pool) *MessageStore {
	return &MessageStore{pool: pool}
}

func (s *MessageStore) List(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, conversation_id, role, content, created_at
		 FROM messages
		 WHERE conversation_id = $1
		 ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("messages: list: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("messages: list scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	if msgs == nil {
		msgs = []Message{}
	}
	return msgs, rows.Err()
}

func (s *MessageStore) Create(ctx context.Context, conversationID, role, content string) (*Message, error) {
	var m Message
	err := s.pool.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, role, content)
		 VALUES ($1, $2, $3)
		 RETURNING id, conversation_id, role, content, created_at`,
		conversationID, role, content,
	).Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("messages: create: %w", err)
	}
	return &m, nil
}
