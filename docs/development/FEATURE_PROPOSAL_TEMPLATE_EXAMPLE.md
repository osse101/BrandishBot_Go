# Feature Proposal — Gamble v1

## 1. Title
**Short name:** Gamble v1  
**Canonical ID:** feature/gamble-v1

## 2. One-line summary
Multiplayer lootbox gamble where users bet lootboxes, the server opens them, and the highest-value participant wins the pooled items.

## 3. Goal(s)
- Implement a competitive gambling minigame involving multiple users.
- Ensure atomic item transfers and strong consistency for lootbox consumption.
- Provide deterministic event flow for chatbot announcements.

## 4. Scope
### In scope
- Start gamble
- Join gamble
- Consume lootboxes from each participant
- Open lootboxes server-side
- Compute winner by item value sum
- Award pooled items
- Announce stages via internal events

### Out of scope
- Marketplace or trade systems
- Non-lootbox item wagering
- Cross-channel or cross-server gambles

## 5. Motivation
Adds a social, high-stakes multiplayer mechanic that integrates existing lootbox and item systems while stressing event, persistence, and concurrency architecture.

## 6. Stakeholders
- Owner: Osse  
- Backend reviewer: TBD  
- Ops: TBD  
- Chatbot UX: TBD  

## 7. References / Constraint Docs
- Architecture constraints: [ARCHITECTURE.md](file:///home/osse1/projects/BrandishBot_Go/docs/ARCHITECTURE.md)
- Database configuration: [DATABASE.md](file:///home/osse1/projects/BrandishBot_Go/docs/DATABASE.md)
- Migrations & schema evolution: [MIGRATIONS.md](file:///home/osse1/projects/BrandishBot_Go/docs/MIGRATIONS.md)
- Security analysis & input validation: [SECURITY_ANALYSIS.md](file:///home/osse1/projects/BrandishBot_Go/docs/SECURITY_ANALYSIS.md)
- Code quality & testing standards: [CODE_QUALITY_RECOMMENDATIONS.md](file:///home/osse1/projects/BrandishBot_Go/docs/development/CODE_QUALITY_RECOMMENDATIONS.md)

## 8. Non-functional Constraints
- **Stack**: Go 1.x, Docker, pgx/v5, PostgreSQL 15+, goose (migrations), Swagger, validator
- **Concurrency**: Heavy during join phase; must avoid deadlocks using ordered row-level locks
- **Observability**: Structured JSON logs + metrics for joins, failures, and completion

## 9. Data model (DDL sketch)
### `gambles`
- `id` UUID PRIMARY KEY  
- `initiator_id` UUID REFERENCES users(user_id)  
- `state` TEXT CHECK (state IN ('Created', 'Joining', 'Opening', 'Completed', 'Refunded'))  
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT NOW()  
- `join_deadline` TIMESTAMPTZ NOT NULL

### `gamble_participants`
- `gamble_id` UUID REFERENCES gambles(id) ON DELETE CASCADE  
- `user_id` UUID REFERENCES users(user_id)  
- `lootbox_ids` UUID[] NOT NULL  
- PRIMARY KEY (`gamble_id`, `user_id`)

### `gamble_opened_items`
- `gamble_id` UUID REFERENCES gambles(id) ON DELETE CASCADE  
- `user_id` UUID REFERENCES users(user_id)  
- `item_id` UUID REFERENCES items(item_id)  
- `value` BIGINT NOT NULL  -- Using BIGINT to avoid overflow
- UNIQUE (`gamble_id`, `user_id`, `item_id`)

**Indexes**:
- `idx_gambles_state` ON gambles(state)
- `idx_gp_gamble_id` ON gamble_participants(gamble_id)
- `idx_goi_gamble_id` ON gamble_opened_items(gamble_id)

## 10. API & Events

### External API
```http
POST /gamble/start
Authorization: X-API-Key: <api_key>
Content-Type: application/json

{
  "user_id": "uuid",
  "lootbox_ids": ["uuid", "uuid"]
}

Response 201:
{
  "gamble_id": "uuid",
  "join_deadline": "2025-11-28T18:00:00Z"
}

---

POST /gamble/{id}/join
Authorization: X-API-Key: <api_key>
Content-Type: application/json

{
  "user_id": "uuid",
  "lootbox_ids": ["uuid", "uuid"]
}

Response 200:
{
  "message": "Successfully joined gamble"
}

---

GET /gamble/{id}
Response 200:
{
  "gamble_id": "uuid",
  "state": "Joining",
  "initiator_id": "uuid",
  "join_deadline": "2025-11-28T18:00:00Z",
  "participants": [
    {
      "user_id": "uuid",
      "username": "user1",
      "lootbox_count": 2
    }
  ]
}
```

### Internal Events
Events emitted for bot integration and statistics tracking:

- `GambleStarted { gamble_id, initiator_id, buy_in_count, join_deadline }`
- `GambleJoined { gamble_id, user_id, lootbox_count }`
- `GambleOpening { gamble_id, participant_count }`
- `GambleCompleted { gamble_id, winner_id, prize_items[], total_value }`
- `GambleRefunded { gamble_id, reason }`

## 11. Business Rules
1. Each participant must supply their own lootboxes (minimum 1, maximum configurable).
2. Lootboxes must be locked and consumed upfront upon joining.
3. No user may join without enough lootboxes in their inventory.
4. Only one active gamble per channel at a time.
5. Join phase ends at `join_deadline`; system transitions automatically.
6. If only initiator joined → full refund to initiator, state set to `Refunded`.
7. Winner is user with highest `sum(opened_items.value)`.
8. Winner receives all pooled items; losers' items are consumed (lost).
9. All item transfers and state updates occur within a single transaction.

## 12. Failure modes & edge cases
- **Concurrent join attempts** → Handled with `SELECT ... FOR UPDATE` row-level locks on `gamble_participants`.
- **Missing lootboxes after join** → Reject join request with 400 Bad Request.
- **Opening stage fails midway** → Entire gamble rolled back within transaction; emit `GambleRefunded` event.
- **Join deadline passed but state not updated** → Background worker transitions state to `Opening`.
- **Database deadlock** → Use ordered locking: lock user inventory in ascending `user_id` order.
- **Duplicate join attempts** → PRIMARY KEY constraint prevents duplicates; return 409 Conflict.

## 13. Security & Authorization
- **Authentication**: All endpoints require `X-API-Key` header (see [SECURITY_ANALYSIS.md](file:///home/osse1/projects/BrandishBot_Go/docs/SECURITY_ANALYSIS.md)).
- **Input validation**:
  - `user_id`: Valid UUID format
  - `lootbox_ids`: Array of valid UUIDs, length 1-10
  - `gamble_id`: Valid UUID format
- **Rate limiting**: Apply per-IP rate limits to prevent spam (10 req/sec, burst 20).
- **Audit logs**: Log all gamble events (start, join, open, award, refund) with correlation IDs.

## 14. Observability
### Logs
- `gamble.started`: `gamble_id`, `initiator_id`, `join_deadline`
- `gamble.join.accepted`: `gamble_id`, `user_id`, `lootbox_count`
- `gamble.join.rejected`: `gamble_id`, `user_id`, `reason`
- `gamble.opening`: `gamble_id`, `participant_count`
- `gamble.completed`: `gamble_id`, `winner_id`, `prize_count`, `total_value`
- `gamble.refunded`: `gamble_id`, `reason`

### Metrics (Prometheus)
- `gambles_started_total` (counter)
- `gambles_joined_total` (counter)
- `gambles_completed_total` (counter)
- `gambles_refunded_total` (counter)
- `gamble_join_duration_seconds` (histogram)
- `gamble_opening_duration_seconds` (histogram)

### Traces
- Span: `POST /gamble/start` → DB transaction
- Span: `POST /gamble/{id}/join` → Lock inventory → Consume lootboxes
- Span: `GambleOpening` → Open all lootboxes → Compute winner → Award items

## 15. Testing Requirements
### Unit Tests
- Service logic: `CreateGamble()`, `JoinGamble()`, `OpenGamble()`
- Value computation: Winner selection with edge cases (ties, single participant)
- Inventory locking: Verify lootboxes are correctly consumed

### Integration Tests
- DB locking: Simulate concurrent joins, verify only one succeeds per user
- Transaction rollback: Inject failure during opening stage, verify refund
- End-to-end flow: Start → Join (multiple users) → Auto-transition → Complete

### Concurrency Tests
- 1000 concurrent join attempts, verify no duplicates and proper error handling
- Deadlock prevention: Lock users in sorted order

### Benchmarks
- Joining throughput: Target 100 joins/sec
- Opening latency: Target < 500ms for 10 participants with 5 lootboxes each

## 16. Migrations
### Migration file
- **File**: `migrations/YYYYMMDDHHMMSS_create_gamble_tables.sql` (goose naming convention)
- **Migration tool**: goose (see [MIGRATIONS.md](file:///home/osse1/projects/BrandishBot_Go/docs/MIGRATIONS.md))

### Up migration
```sql
-- +goose Up
CREATE TABLE gambles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initiator_id UUID NOT NULL REFERENCES users(user_id),
    state TEXT NOT NULL CHECK (state IN ('Created', 'Joining', 'Opening', 'Completed', 'Refunded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    join_deadline TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_gambles_state ON gambles(state);

CREATE TABLE gamble_participants (
    gamble_id UUID REFERENCES gambles(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id),
    lootbox_ids UUID[] NOT NULL,
    PRIMARY KEY (gamble_id, user_id)
);

CREATE INDEX idx_gp_gamble_id ON gamble_participants(gamble_id);

CREATE TABLE gamble_opened_items (
    gamble_id UUID REFERENCES gambles(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id),
    item_id UUID REFERENCES items(item_id),
    value BIGINT NOT NULL,
    UNIQUE (gamble_id, user_id, item_id)
);

CREATE INDEX idx_goi_gamble_id ON gamble_opened_items(gamble_id);
```

### Down migration
```sql
-- +goose Down
DROP TABLE IF EXISTS gamble_opened_items;
DROP TABLE IF EXISTS gamble_participants;
DROP TABLE IF EXISTS gambles;
```

### Backwards compatibility
- Read-only `GET /gamble/{id}` endpoint works before and after migration.
- New tables do not affect existing functionality.

## 17. Deployment
1. **Apply DB migrations**: `goose -dir migrations postgres "postgres://..." up`
2. **Deploy service image**: `docker compose up -d app` or K8s deployment
3. **Monitor metrics**: Watch `gambles_started_total`, `gamble_join_duration_seconds`
4. **Verify health**: `curl http://localhost:8080/health`

### Rollback plan
- If critical bug found: Revert deployment, run `goose down` to remove tables
- No data loss if gambles have not yet started

## 18. Implementation Strategy
### Task Breakdown
| Task | Size | Description |
|------|------|-------------|
| 1. DB schema | S | Create goose migration file with tables and indexes |
| 2. Repository layer | M | `CreateGamble`, `JoinGamble`, `LockLootboxes`, `RecordOpenedItems`, `CompleteGamble` |
| 3. Service layer | L | State machine, join phase logic, opening orchestration, winner computation |
| 4. API handlers | S | `POST /gamble/start`, `POST /gamble/{id}/join`, `GET /gamble/{id}` |
| 5. Event emission | S | Emit events to stats service / bot integration |
| 6. Lootbox opening logic | M | Reuse existing `ProcessLootbox` with transaction support |
| 7. Tests | M | Unit, integration, concurrency tests |
| 8. Background worker | M | Auto-transition gambles from `Joining` to `Opening` at deadline |

**Estimated Total**: 2-3 weeks (1 developer)

### Decision Points
**DP1: Lootbox locking strategy**  
- **Option A**: Lock inventory rows with `SELECT ... FOR UPDATE` (preferred)  
  - ✅ Built-in PostgreSQL support
  - ✅ Automatic deadlock detection
  - ❌ Requires careful lock ordering
- **Option B**: Optimistic CAS on `available_count` column
  - ✅ No explicit locks
  - ❌ Retry logic required
  - ❌ More complex rollback

**Decision**: Option A (row-level locks with ordered locking by `user_id`)

---

**DP2: Opening execution model**  
- **Option A**: Single transaction for entire gamble (preferred)  
  - ✅ Full atomicity and consistency
  - ✅ Simpler rollback logic
  - ❌ Long-running transaction risk
- **Option B**: Per-user transaction with compensation logic
  - ✅ Shorter transactions
  - ❌ Complex compensation on partial failure
  - ❌ Consistency harder to guarantee

**Decision**: Option A (single transaction with timeout monitoring)

---

**DP3: Storage of opened items**  
- **Option A**: Insert into `gamble_opened_items` table (preferred)  
  - ✅ Persisted for audit and debugging
  - ✅ Can query historical gambles
  - ❌ Slight storage overhead
- **Option B**: Temporary in-memory store
  - ✅ Faster processing
  - ❌ Data lost on crash
  - ❌ No audit trail

**Decision**: Option A (persistent storage in `gamble_opened_items`)

### Sequence Diagram (Happy Path)
```
User A                Bot                Service               Database
  |                    |                    |                    |
  | /gamble start ------>------------------>|                    |
  |                    |                    | CREATE gamble ---->|
  |                    |                    | LOCK inventory --->|
  |                    |                    | CONSUME lootboxes->|
  |                    |<-- GambleStarted --|<-------------------|
  |<--- gamble_id -----|                    |                    |
  |                    |                    |                    |
User B                 |                    |                    |
  |                    |                    |                    |
  | /gamble join -------->----------------->|                    |
  |                    |                    | LOCK inventory --->|
  |                    |                    | INSERT participant>|
  |                    |                    | CONSUME lootboxes->|
  |                    |<--- GambleJoined --|<-------------------|
  |<--- success -------|                    |                    |
  |                    |                    |                    |
  |        (join_deadline reached)          |                    |
  |                    |                    |                    |
  |                    | Worker ----------->|                    |
  |                    |                    | UPDATE state='Opening'>
  |                    |<-- GambleOpening --|                    |
  |                    |                    | OPEN lootboxes --->|
  |                    |                    | INSERT opened_items>
  |                    |                    | COMPUTE winner --->|
  |                    |                    | AWARD items ------>|
  |                    |                    | UPDATE state='Completed'>
  |                    |<-- GambleCompleted-|<-------------------|
  |<--- announcement --|                    |                    |
```

## 19. Acceptance Criteria
- **AC1**: Winner receives all pooled items; database state is consistent; losers' inventories correctly decremented.
- **AC2**: Under high concurrency (100 concurrent joins), no duplicate `gamble_participants` rows exist.
- **AC3**: Refund scenario works: if only initiator joined, lootboxes are returned, state set to `Refunded`.
- **AC4**: All events emitted exactly once per gamble lifecycle stage.
- **AC5**: All API endpoints return appropriate HTTP status codes and error messages.
- **AC6**: All business rules (section 11) are enforced and tested.

## 20. Risks & Mitigations
| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| DB deadlocks | High | Medium | Use ordered locking (ascending `user_id`) |
| Long-running open stage | Medium | Medium | Batch lootbox opening, add timeout monitoring |
| Item valuation overflow | Medium | Low | Use `BIGINT` (int64) for value storage |
| Insufficient inventory | Medium | High | Validate inventory before join, return clear error |
| Background worker failure | High | Low | Add idempotency checks, retry logic, alerting |

---

## Template Usage Guide

This document follows the **BrandishBot_Go Feature Design Template**. When creating a new feature proposal:

### Required Sections
1. **Title** (section 1): Provide short name and canonical ID for version control
2. **Goal(s)** (section 3): Define measurable success criteria
3. **Scope** (section 4): Explicitly list what IS and ISN'T included
4. **References** (section 7): Link to project constraint documents (see above for examples)
5. **Non-functional Constraints** (section 8): List stack, concurrency, observability requirements
6. **Data Model** (section 9): Provide DDL with proper types, constraints, and indexes
7. **API & Events** (section 10): Document HTTP endpoints and internal events
8. **Business Rules** (section 11): Enumerate all rules, edge cases, and invariants
9. **Testing Requirements** (section 15): Specify unit, integration, concurrency tests
10. **Migrations** (section 16): Provide goose migration files (up and down)
11. **Acceptance Criteria** (section 19): Define clear, testable success conditions

### Best Practices
- **Link guideline documents**: Always reference [ARCHITECTURE.md](file:///home/osse1/projects/BrandishBot_Go/docs/ARCHITECTURE.md), [SECURITY_ANALYSIS.md](file:///home/osse1/projects/BrandishBot_Go/docs/SECURITY_ANALYSIS.md), etc.
- **Use project conventions**: 
  - goose for migrations (not generic "SQL migrations")
  - pgx/v5 for database access
  - UUID primary keys
  - Structured JSON logging
- **Security by default**: Always include authentication, input validation, and rate limiting
- **Observability first**: Define logs, metrics, and traces upfront
- **Test requirements**: Specify concrete test scenarios, not just "add tests"

### Optional Sections
- **Stakeholders** (section 6): Use if involving multiple teams
- **Sequence Diagram** (section 18): Helpful for complex flows
- **Risks & Mitigations** (section 20): Recommended for high-impact features

### Validation Checklist
Before submitting a feature proposal, verify:
- [ ] All referenced documents are linked with absolute paths
- [ ] Database schema uses proper PostgreSQL types and constraints
- [ ] Security requirements (auth, validation, rate limiting) are specified
- [ ] Migration files include both `up` and `down` SQL
- [ ] Acceptance criteria are specific and testable
- [ ] Observability requirements (logs, metrics, traces) are defined
