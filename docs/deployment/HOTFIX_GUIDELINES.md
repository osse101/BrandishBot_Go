# Hotfix Guidelines: What Can and Cannot Be Hotfixed While Live

This document provides clear guidelines on what changes can be safely hotfixed to a live production environment versus changes that require a full deployment cycle.

## Table of Contents
- [Quick Reference](#quick-reference)
- [Safe to Hotfix](#safe-to-hotfix)
- [Requires Full Deployment](#requires-full-deployment)
- [Hotfix Process](#hotfix-process)
- [Testing Requirements](#testing-requirements)
- [Rollback Procedures](#rollback-procedures)
- [Emergency Hotfix Checklist](#emergency-hotfix-checklist)

---

## Quick Reference

### ✅ Safe to Hotfix (Low Risk)

| Change Type | Risk Level | Downtime | Testing Required |
|------------|------------|----------|------------------|
| **Configuration Changes** | Low | None | Smoke test |
| **Environment Variables** | Low | Container restart (2-5s) | Validation |
| **Log Level Adjustments** | Very Low | None | Log verification |
| **Discord Embed Text** | Very Low | None | Visual check |
| **Error Messages** | Low | None | Error flow test |
| **Feature Flags** | Low | None | Toggle verification |
| **Typo Fixes (UI Text)** | Very Low | None | Visual check |
| **Documentation Updates** | Very Low | None | None |

### ⚠️ Requires Testing on Staging First (Medium Risk)

| Change Type | Risk Level | Why Staging Is Required |
|------------|------------|-------------------------|
| **Bug Fixes (Logic)** | Medium | May have unintended side effects |
| **API Response Changes** | Medium | May break Discord bot or clients |
| **New Command Handlers** | Medium | Needs integration testing |
| **Constants/Rates Changes** | Medium | Impacts game balance |
| **Validation Rules** | Medium | May reject valid input or allow invalid |
| **Cache TTL Changes** | Low-Medium | Performance impact unknown |

### 🛑 Never Hotfix (High Risk)

| Change Type | Risk Level | Why Full Deployment Is Required |
|------------|------------|----------------------------------|
| **Database Migrations** | Critical | Data loss risk, requires backups |
| **Schema Changes** | Critical | Backward compatibility issues |
| **API Signature Changes** | High | Breaking changes for clients |
| **Dependency Updates** | High | Untested compatibility issues |
| **New Dependencies** | High | Build required, compatibility unknown |
| **Refactoring** | Medium-High | Extensive testing needed |
| **Concurrency Changes** | High | Race condition risks |
| **Security Changes** | High | Requires security review |

---

## Safe to Hotfix

These changes have minimal risk and can be deployed directly to production after basic validation.

### 1. Configuration Changes (Non-Breaking)

**Examples:**
- Adjusting timeouts (request timeout, Discord timeout)
- Changing retry counts
- Modifying batch sizes
- Updating webhook URLs

**Why Safe:**
- No code changes required
- Can be reverted by changing config back
- No database impact

**Hotfix Process:**
```bash
# 1. Update .env file
nano .env
# Change: DISCORD_TIMEOUT=30 to DISCORD_TIMEOUT=60

# 2. Restart containers to pick up new env vars
docker compose -f docker-compose.production.yml restart app discord

# 3. Verify
curl http://localhost:8080/healthz
docker compose -f docker-compose.production.yml logs --tail=50 app
```

**Testing Required:**
- ✅ Health check passes
- ✅ Log shows new value loaded
- ✅ Related functionality works (e.g., Discord commands respond)

---

### 2. Log Level Adjustments

**Examples:**
- Changing `LOG_LEVEL` from `info` to `debug` (for troubleshooting)
- Changing back from `debug` to `info` (to reduce log noise)

**Why Safe:**
- No functional changes
- Only affects log output
- Instantly reversible

**Hotfix Process:**
```bash
# 1. Update .env
nano .env
# Change: LOG_LEVEL=info to LOG_LEVEL=debug

# 2. Restart
docker compose -f docker-compose.production.yml restart app

# 3. Verify logs are more verbose
docker compose -f docker-compose.production.yml logs --tail=100 app
```

**Testing Required:**
- ✅ Verify log level change in output

---

### 3. Discord Embed Text / UI Messages

**Examples:**
- Fixing typos in Discord messages
- Improving wording of user-facing messages
- Updating help text

**Why Safe:**
- Cosmetic changes only
- No logic changes
- No database impact

**Files You Can Modify:**
- `internal/discord/embeds.go`
- Discord command descriptions in `internal/discord/cmd_*.go`
- Error messages in `internal/handler/errors.go` (if just text, not logic)

**Hotfix Process:**
```bash
# 1. Fix the typo
git checkout production
nano internal/discord/embeds.go
# Change: "Sucessfully" to "Successfully"

# 2. Commit
git add internal/discord/embeds.go
git commit -m "fix: typo in success message"

# 3. Build and deploy
make deploy-production

# 4. Test in Discord
# Run the command and verify the message is correct
```

**Testing Required:**
- ✅ Visual verification in Discord
- ✅ No errors in logs

---

### 4. Feature Flags (Toggle Existing Features)

**Examples:**
- Enabling/disabling `DEV_MODE`
- Toggling `DISCORD_FORCE_COMMAND_UPDATE`

**Why Safe:**
- Designed to be toggled
- No code changes
- Instant rollback by toggling back

**Hotfix Process:**
```bash
# 1. Update .env
nano .env
# Change: DEV_MODE=false to DEV_MODE=true

# 2. Restart
docker compose -f docker-compose.production.yml restart app discord

# 3. Verify flag is active
# Check logs or test behavior
```

**Testing Required:**
- ✅ Feature behaves as expected with flag on/off
- ✅ No unintended side effects

---

### 5. Error Message Wording

**Examples:**
- Clarifying error messages for users
- Adding more helpful error context
- Fixing grammar in error strings

**Why Safe:**
- No change to error handling logic
- Only changes user-facing text

**Hotfix Process:**
```bash
git checkout production
nano internal/handler/errors.go

# Change:
# "Item not found" → "The item you specified was not found in the database"

git commit -am "fix: improve error message clarity"
make deploy-production
```

**Testing Required:**
- ✅ Trigger the error condition
- ✅ Verify new message appears
- ✅ Error handling logic unchanged

---

## Requires Full Deployment

These changes have higher risk and **must** go through the full staging → production workflow.

### 1. Database Migrations

**Examples:**
- Adding a new table
- Adding/removing columns
- Changing column types
- Creating indexes
- Adding constraints

**Why Not Safe:**
- **Data loss risk**: Wrong migration can corrupt data
- **Rollback complexity**: Requires database restore
- **Backward compatibility**: Old app version may break with new schema
- **Testing required**: Must test migration on staging database first

**Full Deployment Process:**
```bash
# 1. Create migration on develop
git checkout develop
make migrate-create NAME=add_new_feature_table

# 2. Write migration
nano migrations/XXXXXXX_add_new_feature_table.sql

# 3. Test locally
make docker-up
make migrate-up
make test

# 4. Deploy to staging
git checkout staging
git merge develop --no-ff
make deploy-staging
make health-check-staging

# 5. Validate migration on staging
docker compose -f docker-compose.staging.yml exec db psql -U $DB_USER -d $DB_NAME
# Check table was created, data is intact

# 6. If good, deploy to production
git checkout production
git merge staging --no-ff
make deploy-production
```

**Testing Required:**
- ✅ Migration runs successfully on staging
- ✅ Existing data is not affected
- ✅ New schema is correct
- ✅ Application works with new schema
- ✅ Rollback migration tested

> [!CAUTION]
> **Never hotfix migrations**
> 
> Database migrations can cause irreversible data loss. Always test on staging first and have a backup plan.

---

### 2. API Signature Changes

**Examples:**
- Adding a required parameter to an endpoint
- Changing response structure
- Renaming fields in JSON responses
- Changing HTTP status codes

**Why Not Safe:**
- **Breaking changes**: Existing clients (Discord bot, Streamer.bot) may break
- **Integration testing**: Need to test all clients against new API
- **Rollback complexity**: Clients may have already updated

**Full Deployment Process:**
```bash
# 1. Make changes on develop
git checkout develop
# ... modify endpoints ...

# 2. Update API clients (Discord bot, etc.)
# ... update internal/discord/client.go ...

# 3. Test integration locally
make build
make run
# Test Discord commands against new API

# 4. Deploy to staging
git checkout staging
git merge develop --no-ff
make deploy-staging

# 5. Full integration testing
make test-staging
# Manually test all Discord commands

# 6. Deploy to production
git checkout production
git merge staging --no-ff
make deploy-production
```

**Testing Required:**
- ✅ All API clients tested against new signatures
- ✅ Backward compatibility verified (if applicable)
- ✅ Error handling tested
- ✅ Integration tests pass

---

### 3. Dependency Updates

**Examples:**
- Updating Go dependencies (`go get -u`)
- Updating Docker base image version
- Adding new external libraries

**Why Not Safe:**
- **Compatibility issues**: New versions may have breaking changes
- **Build required**: Can't be done with just config changes
- **Security risks**: New dependencies may have vulnerabilities
- **Testing needed**: Behavior may change subtly

**Full Deployment Process:**
```bash
# 1. Update dependencies on develop
git checkout develop
go get -u ./...
go mod tidy

# 2. Run tests
make test
make test-integration

# 3. Build and test locally
make docker-build
make docker-up
# Test all functionality

# 4. Deploy to staging
git checkout staging
git merge develop --no-ff
make deploy-staging

# 5. Extended testing on staging (48+ hours)
make test-staging
# Monitor for crashes, memory leaks, performance issues

# 6. Deploy to production
git checkout production
git merge staging --no-ff
make deploy-production
```

**Testing Required:**
- ✅ All tests pass
- ✅ No new vulnerabilities (`go list -m -json all | nancy sleuth`)
- ✅ No performance degradation
- ✅ Stability testing on staging for 48+ hours

---

### 4. Refactoring / Code Restructuring

**Examples:**
- Moving functions between files
- Renaming internal variables/functions
- Changing function signatures (internal)
- Restructuring packages

**Why Not Safe:**
- **Regression risk**: May introduce subtle bugs
- **Extensive testing**: Need to verify all affected code paths
- **Code review**: Should be reviewed by team
- **Not urgent**: Refactoring is never urgent enough to bypass staging

**Full Deployment Process:**
- **Always** go through develop → staging → production
- **Always** include comprehensive tests
- **Never** rush refactoring into production

---

### 5. Concurrency / Race Condition Fixes

**Examples:**
- Adding mutexes
- Changing goroutine behavior
- Fixing race conditions
- Modifying transaction handling

**Why Not Safe:**
- **Critical bugs**: Race conditions are hard to test
- **Production risk**: May cause deadlocks or panics
- **Testing difficulty**: Race conditions may only appear under load
- **Staging validation**: Must run under realistic load

**Full Deployment Process:**
```bash
# 1. Fix on develop
git checkout develop
# ... add mutex protection ...

# 2. Run race detector
go test -race ./...

# 3. Load testing on staging
git checkout staging
git merge develop --no-ff
make deploy-staging

# 4. Monitor staging under load
# Run load tests, monitor for deadlocks
# Leave running for 24+ hours

# 5. Deploy to production only after stability proven
git checkout production
git merge staging --no-ff
make deploy-production
```

**Testing Required:**
- ✅ Race detector passes (`go test -race`)
- ✅ Load testing on staging
- ✅ No deadlocks observed
- ✅ 24+ hours of stable operation on staging

---

### 6. Security Changes

**Examples:**
- Fixing authentication bypass
- Patching SQL injection
- Updating API key validation
- Fixing XSS vulnerabilities

**Why Not Safe:**
- **Security review needed**: Changes must be reviewed carefully
- **Testing required**: Security fixes can break legitimate use cases
- **Rollback plan**: If fix breaks functionality, need quick rollback
- **Coordination**: May need to rotate API keys, notify users

**Full Deployment Process:**
```bash
# For CRITICAL security issues (active exploit):
# Use Emergency Hotfix Process (see below)

# For NON-CRITICAL security issues:
# 1. Fix on develop with comprehensive tests
git checkout develop
# ... security fix ...

# 2. Security testing
# Test bypass attempts, penetration testing

# 3. Deploy to staging
git checkout staging
git merge develop --no-ff
make deploy-staging

# 4. Verify fix doesn't break legitimate use
# Test all auth flows, API endpoints

# 5. Deploy to production
git checkout production
git merge staging --no-ff
make deploy-production

# 6. Monitor for abuse attempts
docker compose -f docker-compose.production.yml logs -f app | grep -i "security\|auth\|unauthorized"
```

**Testing Required:**
- ✅ Security vulnerability is fixed
- ✅ Legitimate use cases still work
- ✅ No new vulnerabilities introduced
- ✅ Error handling is secure (no info leaks)

> [!WARNING]
> **Critical Security Vulnerabilities**
> 
> If there's an **active exploit** happening in production, use the Emergency Hotfix Process. Otherwise, use full deployment cycle.

---

## Hotfix Process

### Standard Hotfix (Safe Changes)

Use this for changes that are [Safe to Hotfix](#safe-to-hotfix).

```bash
# 1. Create hotfix branch from production
git checkout production
git pull origin production
git checkout -b hotfix/fix-typo-in-message

# 2. Make the change
nano internal/discord/embeds.go
# Fix typo

# 3. Test locally
make build
make run
# Verify fix

# 4. Commit
git add internal/discord/embeds.go
git commit -m "fix: typo in inventory message"

# 5. Merge to production
git checkout production
git merge hotfix/fix-typo-in-message --no-ff -m "hotfix: fix typo in inventory message v1.2.1"
git tag v1.2.1

# 6. Deploy to production
make deploy-production

# 7. Verify in production
curl http://localhost:8080/healthz
# Test in Discord

# 8. Backport to develop and staging
git checkout develop
git merge hotfix/fix-typo-in-message
git push origin develop

git checkout staging
git merge develop
git push origin staging

# 9. Clean up
git branch -d hotfix/fix-typo-in-message
```

---

### Emergency Hotfix (Critical Issues)

Use this **only** for critical production issues that require immediate fixing (e.g., production is down, active security exploit).

> [!CAUTION]
> This process skips staging validation. Only use for true emergencies.

```bash
# 1. Fix directly on production branch
git checkout production
git pull origin production

# 2. Make the fix
# ... emergency fix ...

# 3. Quick local test
make build
make test
# If tests pass, proceed

# 4. Commit and tag
git add .
git commit -m "fix: emergency fix for critical issue"
git tag v1.2.2

# 5. Deploy immediately
make deploy-production

# 6. Monitor closely
docker compose -f docker-compose.production.yml logs -f app
# Watch for errors

# 7. If deployment fails, rollback immediately
make rollback-production

# 8. Backport to other branches ASAP
git checkout develop
git cherry-pick <commit-hash>
git push origin develop

git checkout staging
git merge develop
git push origin staging
```

---

## Testing Requirements

### Smoke Tests (Always Run)

After **any** hotfix, run these smoke tests:

```bash
# 1. Health check
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}

# 2. Check logs for errors
docker compose -f docker-compose.production.yml logs --tail=100 app | grep -i error

# 3. Test key endpoints
curl http://localhost:8080/progression/tree
curl http://localhost:8080/inventory/items

# 4. Discord bot responsiveness
# Open Discord, run a simple command like /inventory
```

---

### Integration Tests (For Logic Changes)

If the hotfix involves any logic changes:

```bash
# Run integration tests
make test-staging  # Against production (careful!)

# Or run specific package tests
go test ./internal/handler -v
go test ./internal/user -v
```

---

### Manual Tests (For UI/Discord Changes)

If the hotfix changes user-facing behavior:

```yaml
# Test Checklist:
- [ ] Open Discord
- [ ] Run affected command (e.g., /inventory if inventory message changed)
- [ ] Verify new behavior is correct
- [ ] Test edge cases (empty inventory, error conditions)
- [ ] Check for errors in production logs
- [ ] Verify old functionality still works
```

---

## Rollback Procedures

If a hotfix causes issues in production, rollback immediately.

### Quick Rollback (Automated)

```bash
# Using rollback script
make rollback-production

# Follow prompts:
# 1. Select previous version (e.g., v1.2.0)
# 2. Confirm rollback
# 3. Wait for health checks
```

### Manual Rollback

If automated rollback fails:

```bash
# 1. List available versions
docker images brandishbot

# 2. Set target version
export DOCKER_IMAGE_TAG=v1.2.0

# 3. Restart with previous image
docker compose -f docker-compose.production.yml up -d --no-deps app discord

# 4. Verify
curl http://localhost:8080/healthz
```

### Database Rollback

If the hotfix included a migration (which it shouldn't for a hotfix):

```bash
# 1. Rollback app containers
make rollback-production

# 2. Rollback database migration
docker compose -f docker-compose.production.yml exec db sh
# In container:
goose -dir migrations postgres "postgres://$DB_USER:$DB_PASSWORD@localhost:5432/$DB_NAME?sslmode=disable" down

# Or restore from backup
docker compose -f docker-compose.production.yml exec -i db psql -U $DB_USER -d $DB_NAME < backups/backup_production_TIMESTAMP.sql
```

> [!CAUTION]
> Database rollbacks can cause data loss. Only rollback the database if the hotfix corrupted data.

---

## Emergency Hotfix Checklist

Use this checklist when you need to hotfix production in an emergency:

### Pre-Hotfix
- [ ] Severity assessment: Is this truly an emergency?
- [ ] Impact: What's broken? How many users affected?
- [ ] Can we rollback instead of hotfixing forward?
- [ ] Do we have a recent backup?

### During Hotfix
- [ ] Create hotfix branch (or work directly on production for true emergency)
- [ ] Make minimal changes - fix ONLY the critical issue
- [ ] Test locally (even if just 30 seconds of manual testing)
- [ ] Commit with clear message describing the issue and fix
- [ ] Tag the release
- [ ] Deploy to production
- [ ] Monitor logs in real-time for 5 minutes

### Post-Hotfix
- [ ] Verify fix resolves the issue
- [ ] Check for new errors in logs
- [ ] Run smoke tests
- [ ] Notify team of hotfix
- [ ] Document what happened
- [ ] Backport fix to develop and staging
- [ ] Schedule proper fix with full testing if this was a quick patch

### If Hotfix Fails
- [ ] Rollback immediately
- [ ] Restore database backup if needed
- [ ] Investigate root cause
- [ ] Develop proper fix with full testing
- [ ] Deploy through normal staging → production flow

---

## Decision Flow Chart

```mermaid
graph TD
    Start[Change Needed] --> Emergency{Is Production Down?}
    
    Emergency -->|Yes| Critical[Critical Emergency]
    Emergency -->|No| Type{What Type of Change?}
    
    Critical --> EmergencyFix[Emergency Hotfix Process]
    EmergencyFix --> Deploy[Deploy Immediately]
    Deploy --> MonitorClose[Monitor Closely]
    MonitorClose --> BackportLater[Backport to Develop/Staging Later]
    
    Type -->|Config/Env Var| SafeHotfix[Safe to Hotfix]
    Type -->|UI Text/Typo| SafeHotfix
    Type -->|Log Level| SafeHotfix
    
    Type -->|Bug Fix| Staging[Test on Staging First]
    Type -->|Logic Change| Staging
    Type -->|New Feature| Staging
    
    Type -->|Database Migration| FullDeploy[Full Deployment Required]
    Type -->|API Changes| FullDeploy
    Type -->|Dependencies| FullDeploy
    Type -->|Refactoring| FullDeploy
    Type -->|Concurrency Fix| FullDeploy
    
    SafeHotfix --> HotfixBranch[Create Hotfix Branch]
    HotfixBranch --> QuickTest[Quick Local Test]
    QuickTest --> DirectDeploy[Deploy to Production]
    DirectDeploy --> Verify[Smoke Test]
    Verify --> Backport[Backport to Dev/Staging]
    
    Staging --> DevBranch[Fix on Develop]
    DevBranch --> MergeStaging[Merge to Staging]
    MergeStaging --> TestStaging[Test on Staging]
    TestStaging --> StagingOK{Staging OK?}
    StagingOK -->|No| FixIssues[Fix Issues]
    FixIssues --> MergeStaging
    StagingOK -->|Yes| MergeProd[Merge to Production]
    MergeProd --> DeployProd[Deploy Production]
    
    FullDeploy --> DevBranch
```

---

## Summary

### Golden Rules

1. **When in doubt, use staging** - If you're not 100% sure it's safe, test on staging first
2. **Never hotfix migrations** - Database changes always require full deployment
3. **Document everything** - Leave a clear commit message and tag
4. **Monitor after hotfix** - Watch logs for 5-10 minutes after deployment
5. **Have a rollback plan** - Know how to undo your change before deploying

### Safe Hotfix Criteria

A change is safe to hotfix if **ALL** of these are true:

- ✅ No database changes
- ✅ No API signature changes
- ✅ No new dependencies
- ✅ Easily reversible (config change or simple code edit)
- ✅ Low blast radius (affects specific feature, not core functionality)
- ✅ Can be manually tested in 30 seconds
- ✅ No security implications

If **ANY** of these are false, go through staging first.

---

## Related Documentation

- [Production Deployment Strategy](file:///home/osse1/projects/BrandishBot_Go/docs/deployment/PRODUCTION_STRATEGY.md) - Running staging and production on same server
- [Deployment Workflow](file:///home/osse1/projects/BrandishBot_Go/docs/deployment/DEPLOYMENT_WORKFLOW.md) - Complete deployment procedures
- [Rollback Procedures](file:///home/osse1/projects/BrandishBot_Go/docs/deployment/DEPLOYMENT_WORKFLOW.md#rollback-procedures) - Detailed rollback steps
- [Environment Setup](file:///home/osse1/projects/BrandishBot_Go/docs/deployment/ENVIRONMENTS.md) - Environment configuration guide
