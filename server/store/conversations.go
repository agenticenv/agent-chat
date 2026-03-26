package store

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

type ConversationStore struct {
	pool *pgxpool.Pool
}

func NewConversationStore(pool *pgxpool.Pool) *ConversationStore {
	return &ConversationStore{pool: pool}
}

func (s *ConversationStore) List(ctx context.Context) ([]Conversation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, created_at FROM conversations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("conversations: list: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("conversations: list scan: %w", err)
		}
		convs = append(convs, c)
	}
	if convs == nil {
		convs = []Conversation{}
	}
	return convs, rows.Err()
}

func (s *ConversationStore) Create(ctx context.Context, title string) (*Conversation, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`INSERT INTO conversations (title) VALUES ($1) RETURNING id, title, created_at`,
		title,
	).Scan(&c.ID, &c.Title, &c.CreatedAt)
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