package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var safeDBName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Connect opens a pgx pool to DATABASE_URL. If the target database is missing (e.g. Postgres
// volume was created before init-db.sql added CREATE DATABASE), it creates the database using
// the maintenance database `postgres` with the same credentials.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := tryPool(ctx, databaseURL)
	if err == nil {
		return pool, nil
	}
	if !isDatabaseDoesNotExist(err) {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := ensureDatabase(ctx, databaseURL); err != nil {
		return nil, fmt.Errorf("db: ensure database: %w", err)
	}
	pool, err = tryPool(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	return pool, nil
}

func tryPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func isDatabaseDoesNotExist(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "3D000" {
		return true
	}
	// Some pgx paths wrap the error; message still contains SQLSTATE.
	return strings.Contains(err.Error(), "3D000")
}

func ensureDatabase(ctx context.Context, databaseURL string) error {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return err
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return fmt.Errorf("no database name in URL path")
	}
	if !safeDBName.MatchString(dbName) {
		return fmt.Errorf("database name %q is not a safe identifier", dbName)
	}
	u.Path = "/postgres"
	adminPool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		return fmt.Errorf("connect to maintenance db postgres: %w", err)
	}
	defer adminPool.Close()

	var exists bool
	if err := adminPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	if exists {
		return nil
	}
	// dbName validated by safeDBName — safe for simple identifier in CREATE DATABASE
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}
