package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool interface for database connection pool operations
type Pool interface {
	Ping(ctx context.Context) error
	Close()
}

// NewPool creates a new PostgreSQL connection pool
func NewPool(connString string, maxConns int, maxIdle, maxLife time.Duration) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	config.MaxConns = int32(maxConns)
	config.MinConns = 2 // Keeping min conns at 2
	config.MaxConnLifetime = maxLife
	config.MaxConnIdleTime = maxIdle

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to the database")
	return pool, nil
}
