# Project Roadmap & Infrastructure Proposal

**Date:** 2025-11-25
**Status:** Draft

## 1. Available Libraries & Tech Stack Recommendations

Based on the current `go.mod` and project structure (REST API with Postgres), here are recommended libraries to enhance the project.

### Core & Utilities
- **Validation**: `github.com/go-playground/validator/v10`
  - *Why*: Robust struct validation for incoming API requests.
- **API Documentation**: `github.com/swaggo/swag` + `github.com/swaggo/http-swagger`
  - *Why*: Auto-generate Swagger/OpenAPI documentation from comments. Essential for a REST API.
- **Configuration**: `github.com/spf13/viper` (Optional)
  - *Why*: If config complexity grows (watching files, remote config). Currently `godotenv` is fine for simple setups.

### Database & Migrations
- **Migrations**: `github.com/golang-migrate/migrate/v4` or `github.com/pressly/goose/v3`
  - *Why*: Robust migration management. `goose` is Go-native and simpler; `golang-migrate` is very popular.
### Caching
- **None required**: As per requirements, no external caching layer (Redis) is needed at this stage.

### Observability
- **Metrics**: `github.com/prometheus/client_golang`
  - *Why*: Expose standard Go metrics (GC, goroutines) and custom business metrics (items crafted, trades made).
- **Tracing**: `go.opentelemetry.io/otel`
  - *Why*: Trace requests across services (database, external APIs).

### Bot Integration (Discord)
- **Architecture**: Separate Service / Container.
  - *Strategy*: The core Go application will act as the "Game Engine" API. A separate Discord Bot service (Node.js, Python, or another Go binary) will handle Discord Gateway events and communicate with this API.
  - *Communication*: REST API or gRPC between Bot and Engine.
  - *Library*: `github.com/bwmarrin/discordgo` (if building the bot in Go).

## 2. Feature Roadmap

**Context**: This game is designed for a **Live Chatroom** environment.

### Phase 1: Foundation & Stability (Current Focus)
- [ ] **API Documentation**: Implement Swagger/OpenAPI.
- [ ] **Comprehensive Testing**: Increase unit test coverage > 80%. Add integration tests for all endpoints.
- [ ] **Structured Logging**: Ensure all logs are JSON in production.
- [ ] **Health Checks**: Add `/healthz` and `/readyz`.

### Phase 2: Core Gameplay & Progression
- [ ] **Progression System (Locked Content)**:
  - *Core Mechanic*: Most features (Crafting, Trading, specific commands) start locked.
  - *Unlocks*: Features unlock based on player level, achievements, or specific quest completion.
- [ ] **Inventory Management**: Advanced filtering, sorting.
- [ ] **Crafting System**: Recipe discovery, time-based crafting.
- [ ] **Economy**: Global Marketplace, NPC Shops.
- [ ] **Lootboxes**: Probability-based item generation.

### Phase 3: Social & Class System
- [ ] **Class System**: Players choose or earn classes.
  - *Class Bonuses*: Each class provides unique abilities/bonuses (inherent diversity benefit).
  - *Party Mechanics*: Two bonus approaches to evaluate:
    - **Camaraderie Bonus**: Bonuses for partying with same class (teamwork/synergy).
    - **Diversity Bonus**: Bonuses for partying with different classes (already provides individual class benefits).
  - *Recommendation*: Start with **Camaraderie** to encourage class identity and specialization.
- [ ] **Leaderboards**: Monthly rankings for wealth, crafting, etc.
- [ ] **Achievements**: System to track milestones and unlock content.
- [ ] **Trading**: Secure P2P item trading.

### Phase 4: Advanced Tech
- [ ] **Real-time Events**: WebSocket integration for live updates (e.g., "Market crash!", "Raid started!").
- [ ] **Admin Dashboard**: Web UI for game masters to view stats and manage users.

---

## Project Milestones

### Milestone 1: Production-Ready Build
**Goal**: Deploy BrandishBot_Go in parallel to the current system for real-world testing.

**Requirements**:
- [ ] Docker Compose setup for production-like environment.
- [ ] Health checks (`/healthz`, `/readyz`) implemented.
- [ ] Structured JSON logging for production.
- [ ] Database migration tooling (goose or golang-migrate).
- [ ] API documentation (Swagger).
- [ ] Comprehensive test coverage (>80%).
- [ ] Load testing to ensure parity with current system.

**Benefits**: 
- Validate performance and correctness with real user data.
- Identify edge cases and bottlenecks early.
- Build confidence before full migration.

**Recommendation**: This should be the **immediate next milestone** after roadmap approval. Focus on infrastructure, observability, and testing.

---

### Milestone 2: Progression System (Go-Live)
**Goal**: Implement the locked content/progression system to enable live deployment.

**Requirements**:
- [ ] **User Progression Table**: Track player level, XP, unlocked features.
- [ ] **Feature Flags**: Database-driven feature unlocks (crafting, trading, etc.).
- [ ] **API Endpoints**: Check if feature is unlocked for user.
- [ ] **Admin Tools**: Unlock features manually for testing/events.
- [ ] **Discord Bot Integration**: Commands respect progression locks.

**Benefits**:
- Control feature rollout and player experience.
- Create a sense of achievement and engagement.
- Gate complex systems until players are ready.

**Recommendation**: This is the **core differentiator** for the live game. Should be prioritized after Milestone 1 is stable.

---

### Milestone 3: Full Migration from Old System
**Goal**: Migrate all functionality from the legacy Streamerbot system.

**Context**: The old system (likely C# or Python-based) needs to be fully replaced by BrandishBot_Go.

**Requirements**:
- [ ] **Feature Parity**: Audit old system to ensure all commands/features exist in new system.
- [ ] **Data Migration Scripts**: Migrate user inventories, stats, economy data.
- [ ] **Discord Bot**: Complete Discord integration (separate service communicating with API).
- [ ] **Regression Testing**: Validate all migrated features work identically.
- [ ] **User Communication**: Announce migration with changelog.
- [ ] **Rollback Plan**: Keep old system on standby for 2-4 weeks.

**Benefits**:
- Single codebase to maintain.
- Better performance and scalability.
- Modern tech stack for future development.

**Recommendation**: This is the **final milestone**. Only execute after Milestones 1 and 2 are production-proven. Suggested timeline: 2-3 months after Milestone 2 goes live.

**Next Steps for Migration**:
1. Audit the old Streamerbot repository to create a feature checklist.
2. Identify any features NOT yet in BrandishBot_Go.
3. Create a data migration plan (schema mapping, validation).

## 3. Dev/Stage/Prod Environment Prep

### Environment Strategy

| Feature | Development | Staging | Production |
|---------|-------------|---------|------------|
| **Infrastructure** | Local Docker Compose | Single VM / Small K8s Cluster | Managed K8s (EKS/GKE) or Cloud Run |
| **Database** | Local Postgres Container | Managed Postgres (Small instance) | Managed Postgres (HA, Backups) |
| **Config** | `.env` file | Env Vars (CI/CD injected) | Secret Manager / Encrypted Env Vars |
| **Logging** | Text/Console (Human readable) | JSON (Aggregated) | JSON (Aggregated + Alerting) |
| **Debug** | Enabled (Pprof) | Limited | Disabled |

### CI/CD Pipeline (GitHub Actions Recommendation)

1.  **Pull Request Workflow**:
    -   Lint (`golangci-lint`).
    -   Unit Tests (`go test -race`).
    -   Build Check (ensure it compiles).

2.  **Merge to Main Workflow**:
    -   Run Integration Tests.
    -   Build Docker Image.
    -   Push to Container Registry (GHCR/DockerHub).
    -   Deploy to **Staging**.

3.  **Release Workflow (Tag)**:
    -   Promote Staging Image to **Production**.
    -   Run Database Migrations.

### Immediate Next Steps for Infrastructure
1.  **Dockerize**: Ensure `Dockerfile` is optimized (multistage build).
2.  **Compose**: Update `docker compose.yml` to include Prometheus/Grafana for local observability.
3.  **CI**: Create `.github/workflows/ci.yml`.
