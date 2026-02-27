# Worker Pool Architecture

The `internal/worker` package implements a centralized worker pool system for handling background tasks and scheduled operations in BrandishBot.

## Overview

The system uses a `Pool` struct to manage a set of worker goroutines that process `Job` interfaces from a queue. This design allows for:

- Efficient background processing
- Graceful shutdown handling
- Centralized error logging
- Scalable concurrency control

## Core Components

### `Pool` Struct

The `Pool` manages the worker goroutines and the job queue.

```go
type Pool struct {
    workers  int
    jobQueue chan Job
    wg       sync.WaitGroup
    quit     chan struct{}
}
```

- **workers**: Number of concurrent worker goroutines.
- **jobQueue**: Channel for buffering incoming jobs.
- **wg**: WaitGroup for ensuring all workers finish before shutdown.
- **quit**: Channel for signaling shutdown.

### `Job` Interface

All tasks submitted to the pool must implement the `Job` interface:

```go
type Job interface {
    Process(ctx context.Context) error
}
```

- **Process**: The method executed by a worker. It receives a context and returns an error. Errors are logged by the worker but do not stop the pool.

## Usage

### Initialization

Initialize the pool with the desired number of workers and queue size:

```go
pool := worker.NewPool(numWorkers, queueSize)
pool.Start()
```

### Enqueueing Jobs

Submit jobs to the pool:

```go
pool.Enqueue(myJob)
```

Or with context cancellation support (blocks until enqueued or context done):

```go
err := pool.EnqueueContext(ctx, myJob)
```

### Shutdown

Gracefully stop the pool:

```go
pool.Stop() // Closes queue, waits for workers to finish current jobs
```

## Available Workers

The package includes several specialized workers for specific domains. Each worker implements its own `Start()` method to begin scheduling or processing, and a `Shutdown(ctx)` method for graceful termination.

### 1. Daily Reset Worker (`DailyResetWorker`)

- **File**: `daily_reset_worker.go`
- **Purpose**: Handles daily reset tasks at 00:00 UTC+7.
- **Tasks**: Resets daily limits (e.g., Job XP caps), clears temporary data.
- **Logic**: Uses a two-stage timer approach (long-range standby vs. final approach) to efficiently sleep until the next reset window. Publishes `daily_reset_completed` events.

### 2. Weekly Reset Worker (`WeeklyResetWorker`)

- **File**: `weekly_reset_worker.go`
- **Purpose**: Handles weekly reset tasks.
- **Tasks**: Resets weekly quotas (e.g., Weekly Quests), processes weekly rewards.
- **Logic**: Calculates time until next Monday 00:00 UTC and schedules execution. Uses a mutex-protected timer to allow safe rescheduling.

### 3. Expedition Worker (`ExpeditionWorker`)

- **File**: `expedition_worker.go`
- **Purpose**: Manages expedition lifecycle.
- **Tasks**: Processes expedition progress, handles completion events, distributes rewards.
- **Logic**: Subscribes to `expedition.started` events and schedules execution after the join deadline.

### 4. Gamble Worker (`GambleWorker`)

- **File**: `gamble_worker.go`
- **Purpose**: Manages gambling sessions.
- **Tasks**: Handles gamble timeouts, resolves active gambles if stuck.
- **Logic**: Subscribes to `gamble.started` events and schedules execution after the join deadline.

### 5. Subscription Worker (`SubscriptionWorker`)

- **File**: `subscription_worker.go`
- **Purpose**: Manages subscription statuses.
- **Tasks**: Checks for expired subscriptions, verifies status with external APIs (Twitch/YouTube).
- **Logic**: Runs a periodic check (default 6 hours). Marks expired subscriptions as `expired` locally, then requests verification from the external service. Uses rate limiting to prevent API flooding.

## Base Worker

A `BaseWorker` struct is available in `base.go` to provide common functionality for specialized workers, such as timer management (`startTimer`, `stopTimer`), locking, and logging.
