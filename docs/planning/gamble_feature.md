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
- Architecture constraints: link_here
- Transactions & DB: link_here
- Naming guidelines: link_here
- Security: link_here

## 8. Non-functional Constraints
- Stack: go, docker, pgx, postgresql, goose, swagger, validator
- Concurrency: heavy during join phase; must avoid deadlocks
- Observability: structured logs + metrics for joins, failures, and completion

## 9. Data model (DDL sketch)
### gambles
- id UUID PK  
- initiator_id UUID  
- state TEXT (Created, Joining, Opening, Completed, Refunded)  
- created_at TIMESTAMPTZ  
- join_deadline TIMESTAMPTZ  

### gamble_participants
- gamble_id UUID FK  
- user_id UUID  
- lootbox_ids UUID[]  
- PRIMARY KEY (gamble_id, user_id)

### gamble_opened_items
- gamble_id UUID  
- user_id UUID  
- item_id UUID  
- value INT  
- UNIQUE (gamble_id, user_id, item_id)

Indexes:
- idx_gp_gamble_id
- idx_goi_gamble_id

## 10. API & Events

### External API
```
POST /gamble/start
Body: { user_id, lootbox_ids[] }
Returns: { gamble_id }

POST /gamble/{id}/join
Body: { user_id, lootbox_ids[] }
Returns 200

GET /gamble/{id}
Returns public state & participants
```

### Internal Events
- GambleStarted { gamble_id, buy_in }
- GambleJoined { gamble_id, user_id }
- GambleOpening { gamble_id }
- GambleCompleted { gamble_id, winner_id, prize_items[] }
- GambleRefunded { gamble_id }

## 11. Business Rules
1. Each participant must supply their own lootboxes.
2. Lootboxes must be locked and consumed upfront upon joining.
3. No user may join without enough lootboxes.
4. Only one active gamble per channel.
5. Join phase ends at deadline; system transitions automatically.
6. If only initiator joined → full refund to initiator.
7. Winner is user with highest sum(item.value).
8. Winner receives pooled items; losers lose their items.

## 12. Failure modes & edge cases
- Concurrent join attempts → must be handled with DB row-level locks.
- Missing lootboxes after join → reject join request.
- Opening stage fails midway → entire gamble rolled back; retry transaction.

## 13. Security & Authorization
- Only authenticated users can start/join.
- Rate-limit gamble start attempts.
- Audit logs: start, join, open, award.

## 14. Observability
- Logs: join accepted/rejected, lock failures, prize awarded.
- Metrics: gambles_started, gambles_joined, gambles_refunded.
- Spans: start→join→open→complete.

## 15. Testing Requirements
- Unit tests: service logic, value computation.
- Integration tests: DB locking, join concurrency.
- Concurrency: 1000 join attempts with only one row per user.
- Benchmarks: joining throughput.

## 16. Migrations
- goose migration: 20251128_create_gamble_tables.sql
- Backwards compatible with read-only `GET /gamble/{id}`.

## 17. Deployment
- Apply DB migrations first.
- Deploy service image.
- Monitor metrics for join errors.

## 18. Implementation Strategy
### Task Breakdown
1. **DB schema** (S)  
   New tables + goose migration.
2. **Repository layer** (M)  
   Methods: CreateGamble, JoinGamble(tx), LockLootboxes, RecordOpenedItems, CompleteGamble.
3. **Service layer** (L)  
   - State machine  
   - Join phase logic  
   - Opening stage orchestration  
4. **API handlers** (S)  
   `/gamble/start`, `/gamble/{id}/join`
5. **Event emission** (S)
6. **Logic for lootbox opening** (M)
7. **Tests** (M)

### Decision Points
**DP1: Lootbox locking strategy**  
- Option A: Lock rows with `locked_by` column (preferred)  
- Option B: Optimistic CAS on `available_count`

**DP2: Opening execution model**  
- Option A: Single transaction (preferred for atomicity)  
- Option B: Per-user transaction with compensation logic

**DP3: Storage of opened items**  
- Option A: Insert into gamble_opened_items table (preferred)  
- Option B: Temporary in-memory store

### Sequence (Happy Path)
1. POST /gamble/start → create gamble  
2. POST /gamble/{id}/join → lock and consume lootboxes  
3. join_deadline reached → transition to Opening  
4. Open lootboxes → compute winner  
5. Award pooled items → emit event  
6. Update state → Completed

## 19. Acceptance Criteria
- AC1: Winner receives pooled items; DB state consistent.
- AC2: Under high concurrency, no duplicate joins.
- AC3: Refund scenario works with zero participants.
- AC4: All events emitted once.

## 20. Risks
- DB deadlocks → use ordered locking  
- Long-running open stage → batch in chunks  
- Item valuation overflow → int64

