# BrandishBot_Go
Chat Bot Backend in Go

## Project Structure
- `cmd/`: Application entry points
- `internal/`: Private application code
    - `user/`: User service and logic
    - `stats/`: Statistics service
    - `database/`: Database implementations
- `scripts/`: Utility scripts for setup and testing

## Setup
1. Run `scripts/setup_env.sh` to initialize the environment and database.
2. Run `go run cmd/app/main.go` to start the server.

## Testing
Run `scripts/run_tests.sh` to execute all tests.
- Results are logged to `logs/test_results.txt`.
- Concurrency tests are included to ensure thread safety.
