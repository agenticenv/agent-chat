package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conversation run progress status values (conversations.status).
const (
	ConversationStatusIdle      = "idle"
	ConversationStatusRunning   = "running"
	ConversationStatusCompleted = "completed"
	ConversationStatusFailed    = "failed"
)

type Conversation struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AgentStreamID   string    `json:"agentStreamId,omitempty"`
	LastEventOffset int64     `json:"lastEventOffset,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type ConversationStore struct {
	pool *pgxpool.Pool
}

func NewConversationStore(pool *pgxpool.Pool) *ConversationStore {
	return &ConversationStore{pool: pool}
}

func (s *ConversationStore) List(ctx context.Context) ([]Conversation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, status, created_at
		 FROM conversations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("conversations: list: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.Status, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("conversations: list scan: %w", err)
		}
		convs = append(convs, c)
	}
	if convs == nil {
		convs = []Conversation{}
	}
	return convs, rows.Err()
}

// Get returns the full conversation row by id.
func (s *ConversationStore) Get(ctx context.Context, id string) (*Conversation, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, status, agent_stream_id, last_event_offset, created_at
		 FROM conversations WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.Title, &c.Status, &c.AgentStreamID, &c.LastEventOffset, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("conversations: get: %w", err)
	}
	return &c, nil
}

func (s *ConversationStore) Create(ctx context.Context, title string) (*Conversation, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`INSERT INTO conversations (title) VALUES ($1)
		 RETURNING id, title, status, agent_stream_id, last_event_offset, created_at`,
		title,
	).Scan(&c.ID, &c.Title, &c.Status, &c.AgentStreamID, &c.LastEventOffset, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("conversations: create: %w", err)
	}
	return &c, nil
}

func (s *ConversationStore) Update(ctx context.Context, id, title string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE conversations SET title = $1 WHERE id = $2`,
		title, id,
	)
	if err != nil {
		return fmt.Errorf("conversations: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ConversationStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM conversations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("conversations: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ConversationStore) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM conversations WHERE id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("conversations: exists: %w", err)
	}
	return exists, nil
}

// SetRunning marks the conversation as running and stores the agent stream id + offset.
func (s *ConversationStore) SetRunning(ctx context.Context, id, agentStreamID string, offset int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET status = $1, agent_stream_id = $2, last_event_offset = $3
		 WHERE id = $4`,
		ConversationStatusRunning, agentStreamID, offset, id,
	)
	if err != nil {
		return fmt.Errorf("conversations: set running: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateOffset updates last_event_offset while a run is in progress.
func (s *ConversationStore) UpdateOffset(ctx context.Context, id string, offset int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE conversations SET last_event_offset = $1 WHERE id = $2`,
		offset, id,
	)
	if err != nil {
		return fmt.Errorf("conversations: update offset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkCompleted sets status=completed and clears agent_stream_id + last_event_offset.
func (s *ConversationStore) MarkCompleted(ctx context.Context, id string) error {
	return s.markTerminal(ctx, id, ConversationStatusCompleted)
}

// MarkFailed sets status=failed and clears agent_stream_id + last_event_offset.
func (s *ConversationStore) MarkFailed(ctx context.Context, id string) error {
	return s.markTerminal(ctx, id, ConversationStatusFailed)
}

func (s *ConversationStore) markTerminal(ctx context.Context, id, status string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET status = $1, agent_stream_id = '', last_event_offset = 0
		 WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("conversations: mark %s: %w", status, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
