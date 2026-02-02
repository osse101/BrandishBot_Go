package database

// Database Connection Pool Constants
const (
	// DefaultMinConnections is the minimum number of connections to maintain in the pool
	DefaultMinConnections = 2
)

// Error Messages - Database Operations
const (
	ErrMsgFailedToParseConnString     = "failed to parse connection string"
	ErrMsgFailedToCreatePool          = "failed to create connection pool"
	ErrMsgFailedToPingDatabase        = "failed to ping database"
	ErrMsgFailedToBeginTransaction    = "failed to begin transaction"
	ErrMsgFailedToRollbackTransaction = "Failed to rollback transaction"
)

// Log Messages
const (
	LogMsgSuccessfullyConnectedToDatabase = "Successfully connected to the database"
)
