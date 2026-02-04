# 🤖 AI Agent Guidance for BrandishBot_Go

This document provides AI agents with structured guidance on navigating, understanding, and contributing to the BrandishBot_Go project. Use this as your primary reference when working on tasks.

---

## 📚 Quick Navigation by Task Type

| If You're Working On...           | Start Here                                                                    | Journal to Update                                            |
| --------------------------------- | ----------------------------------------------------------------------------- | ------------------------------------------------------------ |
| **New Feature Development**       | [FEATURE_DEVELOPMENT_GUIDE.md](docs/development/FEATURE_DEVELOPMENT_GUIDE.md) | [docs/development/journal.md](docs/development/journal.md)   |
| **Architecture/Design Decisions** | [ARCHITECTURE.md](docs/ARCHITECTURE.md)                                       | [docs/architecture/journal.md](docs/architecture/journal.md) |
| **Writing Tests**                 | [TEST_GUIDANCE.md](docs/testing/TEST_GUIDANCE.md)                             | [docs/testing/journal.md](docs/testing/journal.md)           |
| **Database Operations**           | [DATABASE.md](docs/DATABASE.md), [MIGRATIONS.md](docs/MIGRATIONS.md)          | [docs/development/journal.md](docs/development/journal.md)   |
| **Deployment**                    | [DEPLOYMENT_WORKFLOW.md](docs/deployment/DEPLOYMENT_WORKFLOW.md)              | N/A                                                          |
| **Feature Planning/Proposals**    | [gamble_feature.md](docs/planning/gamble_feature.md) (template example)       | [docs/development/journal.md](docs/development/journal.md)   |
| **Benchmarking**                  | [BENCHMARKING.md](docs/benchmarking/BENCHMARKING.md)                          | [docs/benchmarking/journal.md](docs/benchmarking/journal.md) |
| **API Documentation**             | [API_COVERAGE.md](docs/API_COVERAGE.md)                                       | [docs/development/journal.md](docs/development/journal.md)   |

---

## 🎯 Action-Trigger Guide

| Trigger (Situation)                | Action (Resource/Skill)                                                               |
| ---------------------------------- | ------------------------------------------------------------------------------------- |
| **Need to check/run migrations**   | View **[Goose Skill](.agent/skills/goose/SKILL.md)**                                  |
| **Need to inspect database**       | View **[Postgres Skill](.agent/skills/postgres/SKILL.md)**                            |
| **Need to run tests**              | View **[Testing Skill](.agent/skills/testing/SKILL.md)**                              |
| **Need to deploy/rollback**        | View **[Deployment Skill](.agent/skills/deployment/SKILL.md)**                        |
| **Need to create Discord command** | View **[Discord Skill](.agent/skills/discord/SKILL.md)**                              |
| **Need to modify progression**     | View **[Progression Skill](.agent/skills/progression/SKILL.md)**                      |
| **Need to create a new feature**   | Follow **[Feature Development Guide](docs/development/FEATURE_DEVELOPMENT_GUIDE.md)** |
| **Need to fix a complex bug**      | Check **[Journals](#-journal-files)** for similar past issues                         |
| **Encountering currency/locking**  | Review **[Concurrency Guidelines](#-concurrency-guidelines)**                         |

---

## 📓 Journal Files

**Journals are living documents** where lessons learned, patterns discovered, and best practices are recorded. You should:

1. **Read the relevant journal** before starting work to understand past decisions
2. **Update the journal** after completing work to document any new insights

### Journal Locations

| Journal                                                      | Purpose                                                              | When to Read                           | When to Update                                                                                      |
| ------------------------------------------------------------ | -------------------------------------------------------------------- | -------------------------------------- | --------------------------------------------------------------------------------------------------- |
| [docs/development/journal.md](docs/development/journal.md)   | Development patterns, concurrency, transactions, refactoring         | Building features, fixing bugs         | After discovering patterns, solving tricky bugs                                                     |
| [docs/architecture/journal.md](docs/architecture/journal.md) | System design, scaling, service architecture                         | Design decisions, multi-instance work  | After architectural changes or ADR decisions                                                        |
| [docs/testing/journal.md](docs/testing/journal.md)           | Testing patterns, mocks, coverage strategies                         | Writing tests, debugging test failures | After learning testing lessons                                                                      |
| [docs/tools/journal.md](docs/tools/journal.md)               | Documenting learnings, patterns, and best practices for tools        | Using tools                            | After learning tools lessons                                                                        |
| [docs/benchmarking/journal.md](docs/benchmarking/journal.md) | Documenting learnings, patterns, and best practices for benchmarking | When optimizing                        | After an optimization leads to no improvement or after a new category of optimization is discovered |

### Journal Entry Format

When adding to a journal, use this structure:

```markdown
## YYYY-MM-DD: Title - Brief Description

### Context

What problem were you solving?

### Solution/Pattern

What did you learn or implement?

### Key Learnings

- Bullet points of insights
- Include code examples if helpful

### Impact

What effect does this have on the codebase?

---
```

---

## 🎭 AI Personalities

For specialized AI behaviors, personality configurations, and role-specific prompts, refer to:

📄 **[docs/ai_personalities.md](docs/ai_personalities.md)**

This file contains persona definitions for different task types (debugging, feature development, code review, etc.).

---

## 📁 Project Structure Overview

```MD
BrandishBot_Go/
├── cmd/                    # Entry points (app, discord, setup, debug)
├── internal/               # Core application code
│   ├── database/postgres/  # Repository implementations
│   ├── domain/             # Domain models and constants
│   ├── handler/            # HTTP handlers
│   ├── server/             # Server configuration and routing
│   └── [feature]/          # Feature-specific packages (user, economy, harvest, etc.)
├── configs/                # JSON configuration files
├── migrations/             # Database migration files
├── scripts/                # Deployment and utility scripts (no legacy bash scripts)
├── tests/                  # Integration and staging tests
└── docs/                   # Documentation (see below)
```

### Documentation Structure

```MD
docs/
├── ARCHITECTURE.md         # System architecture overview
├── DATABASE.md             # Database design and schema
├── MIGRATIONS.md           # Migration guide
├── PLAYER_COMMANDS.md      # User-facing commands
├── USAGE.md                # API usage examples
├── architecture/           # Architecture lessons and designs
│   ├── journal.md          # 📓 Architecture journal
│   └── cooldown-service.md # Service design doc
├── development/            # Development guides
│   ├── journal.md          # 📓 Development journal
│   ├── FEATURE_DEVELOPMENT_GUIDE.md  # ** START HERE for features **
│   ├── PROGRESSION_GUIDANCE.md       # ** Deep Dive for Progression **
│   └── CODE_QUALITY_RECOMMENDATIONS.md
├── testing/                # Testing documentation
│   ├── journal.md          # 📓 Testing journal
│   ├── TEST_GUIDANCE.md    # How to write tests
│   └── DATABASE_TESTING.md # Database test patterns
├── planning/               # Feature proposals and roadmaps
│   ├── gamble_feature.md   # Feature proposal template
│   └── PROGRESSION_*.md    # Progression system docs
└── deployment/             # Deployment guides
    ├── DEPLOYMENT_WORKFLOW.md
    └── ENVIRONMENTS.md
```

---

## 🔧 Common Commands (Makefile)

**Always check `make help` for the full list.** Key commands:

```bash
# Development
make build              # Build all binaries to bin/
make run                # Run application from bin/app
make test               # Run tests with coverage and race detection
make unit               # Run unit tests (short mode)
make lint               # Run code linters
make mocks              # Generate mocks (using mockery)
make generate           # Generate sql using sqlc

# Database
make migrate-up         # Run pending migrations
make migrate-down       # Rollback last migration
make migrate-status     # Show migration status
make migrate-create NAME=xyz  # Create new migration

# Docker
make docker-up          # Start services with Docker Compose
make docker-down        # Stop services
make docker-build       # Rebuild images (no cache)
make docker-build-fast  # Rebuild images (with cache)

# Testing
make test-integration   # Run integration tests
make test-staging       # Run staging integration tests
make test-coverage      # Generate HTML coverage report

# Audit & Maintenance (via devtool)
make test-migrations    # Test migration up/down idempotency
make check-deps         # Check required dependencies
make check-db           # Ensure database is running
```

---

## ⚡ AI Agent Best Practices

### Process Management

When running background commands:

1. **Track command IDs** returned by `run_command`
2. **Terminate with IDs** using `send_command_input(..., Terminate: true)`
3. **Clean up at session end** - terminate ALL background processes

```MD
❌ AVOID: Searching for processes by port (unreliable)
✅ CORRECT: Use tracked command ID for cleanup
```

### Debugging Workflow

1. Read relevant journal before investigating
2. Use DEBUG level logging to trace issues
3. Log errors at the boundary with full context
4. Verify fixes with `make test` and `make build`
5. **Document findings** in the appropriate journal

### Security

- **Never expose sensitive information** in code, terminal output, or responses
- Use `.env` files for secrets (never commit `.env`)
- Generic error messages to clients; detailed errors to logs only

---

## 🔒 Concurrency Guidelines

**Key principle**: Use database transactions with `SELECT ... FOR UPDATE`, not application-level locks.

### The Check-Then-Lock Pattern

```go
// Phase 1: Fast rejection (unlocked check)
if onCooldown {
    return ErrOnCooldown{}
}

// Phase 2: Atomic operation
tx.Begin()
lastUsed := SELECT ... FOR UPDATE  // Row lock
if stillOnCooldown { return error }
fn()  // Execute action
UPDATE timestamp
tx.Commit()
```

### Transaction Pattern

```go
tx, err := repo.BeginTx(ctx)
if err != nil { return err }
defer repository.SafeRollback(ctx, tx)

// Operations with tx...

return tx.Commit(ctx)
```

**Full details**: See [docs/development/journal.md](docs/development/journal.md) for concurrency lessons.

---

## 🧪 Testing Checklist

Before submitting changes:

- [ ] `go build ./...` passes
- [ ] `make test` passes (includes `-race` flag)
- [ ] Coverage meets 80% threshold for new code
- [ ] Edge cases tested (empty inputs, boundaries, errors)
- [ ] Mocks reused from existing test files
- [ ] Run `make unit` for quick feedback during development

**Full details**: See [docs/testing/TEST_GUIDANCE.md](docs/testing/TEST_GUIDANCE.md).

---

## 📋 Feature Development Workflow

1. **Plan**: Read requirements, identify integration points
2. **Database**: Create migration if needed
3. **Domain**: Add constants and models
4. **Repository**: Add interface methods and implementations
5. **Service**: Implement business logic
6. **Handler**: Create HTTP endpoint
7. **Routing**: Register in server
8. **Test**: Unit tests (80%+), integration tests
9. **Document**: Update journals with lessons learned

**Full guide**: [docs/development/FEATURE_DEVELOPMENT_GUIDE.md](docs/development/FEATURE_DEVELOPMENT_GUIDE.md)

---

## The Right Tool for Go Code

1. **sed** → Simple text files, one-liners
2. **replace_file_content** → Single contiguous block edits
3. **multi_replace_file_content** → Multiple block edits

---

## 🆘 Troubleshooting Quick Reference

| Issue                      | Solution                                            |
| -------------------------- | --------------------------------------------------- |
| "database does not exist"  | Check `.env.example` DB_NAME, ensure migrations ran |
| Mock type mismatch         | Check interface for exact return types              |
| Race condition             | Use `SELECT ... FOR UPDATE` in transaction          |
| Test goroutine leak        | Add `time.Sleep` before check, use tolerance        |
| Build fails after refactor | Search for old field/type names with grep           |

---

## 📞 Escalation

When stuck or unsure:

1. **Search journals** for similar past issues
2. **Check existing tests** for usage patterns
3. **Document the problem** in the relevant journal, to be updated upon resolution
4. **Ask the user** for clarification or guidance

---

~This guidance was last updated: January 2026~
