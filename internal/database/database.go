package database

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
		return nil, fmt.Errorf("%s: %w", ErrMsgFailedToParseConnString, err)
	}

	if maxConns > math.MaxInt32 {
		maxConns = math.MaxInt32
	}
	config.MaxConns = int32(maxConns)
	config.MinConns = DefaultMinConnections
	config.MaxConnLifetime = maxLife
	config.MaxConnIdleTime = maxIdle

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrMsgFailedToCreatePool, err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrMsgFailedToPingDatabase, err)
	}

	slog.Default().Info(LogMsgSuccessfullyConnectedToDatabase)
	return pool, nil
}
