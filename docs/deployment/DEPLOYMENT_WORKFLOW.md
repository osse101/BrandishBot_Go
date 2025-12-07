# BrandishBot Deployment Workflow

This document describes the complete deployment workflow for BrandishBot from development to production.

## Table of Contents
- [Git Branch Strategy](#git-branch-strategy)
- [Build Lifecycle](#build-lifecycle)
- [Deployment Procedures](#deployment-procedures)

- [Rollback Procedures](#rollback-procedures)
- [Docker Image Management](#docker-image-management)
- [Downtime Windows](#downtime-windows)
- [Hotfix Process](#hotfix-process)

---

## Docker Image Management

You can push your Docker images to a registry (like Docker Hub) to persist staging and production builds for future reference or rollbacks.

### Configuration

1. Log in to your registry:
   ```bash
   docker login
   ```
2. Set your username/registry in `.env`:
   ```bash
   DOCKER_USER=yourusername
   # Optional: DOCKER_IMAGE_NAME=brandishbot (default)
   ```

### Pushing Images

After a successful deployment (or build), you can push the image:

```bash
# Push staging image
make push-staging

# Push production image
make push-production
```

This will tag and push:
- `yourusername/brandishbot:v1.2.3` (Version tag)
- `yourusername/brandishbot:latest-staging` (Environment tag)

---

## Git Branch Strategy

### Branch Structure

```mermaid
gitGraph
    commit id: "v1.0.0"
    branch develop
    checkout develop
    commit id: "feat: add feature A"
    commit id: "feat: add feature B"
    branch staging
    checkout staging
    merge develop tag: "v1.1.0-rc1"
    commit id: "staging tests"
    branch production
    checkout production
    merge staging tag: "v1.1.0"
    checkout develop
    commit id: "feat: add feature C"
```

### Branch Definitions

| Branch | Purpose | Protected | Deployed To |
|--------|---------|-----------|-------------|
| `develop` | Active development, feature integration | Yes | Local dev only |
| `staging` | Pre-release testing, QA validation | Yes | Staging server |
| `production` | Production-ready code | Yes | Production server |
| `master` | Legacy/historical (keep for compatibility) | Yes | Not deployed |


## Server-Side Deployment (Remote)

This method is for servers that **do not build code** but instead pull pre-built images from the registry.

### 1. Server Requirements

The server only needs **Docker** and these specific files:

| File | Purpose |
|------|---------|
| `.env` | Environment secrets and config |
| `scripts/deploy_remote.sh` | Main control script |
| `scripts/health-check.sh` | Used for validation (optional) |
| `docker-compose.staging.yml` | Config for staging |
| `docker-compose.production.yml` | Config for production |

**You do NOT need:** Source code, Go compiler, `Makefile`, or migrating tools (built into the image).

### 2. Usage

Use `deploy_remote.sh` to manage the lifecycle:

```bash
# 1. FULL DEPLOY (Pull -> Restart -> Prune -> Health Check)
./scripts/deploy_remote.sh staging
./scripts/deploy_remote.sh production v1.2.0

# 2. STARTUP (Just start containers)
./scripts/deploy_remote.sh staging latest start

# 3. TEARDOWN (Stop containers)
./scripts/deploy_remote.sh staging latest stop

# 4. PULL ONLY (Pre-fetch images)
./scripts/deploy_remote.sh production v1.3.0 pull
```

### Branch Protection Rules

> [!IMPORTANT]
> Configure these protection rules in your Git hosting:
> - `staging`: Require PR from `develop`, require status checks
> - `production`: Require PR from `staging`, require manual approval
> - `develop`: Require PR from feature branches

---

## Build Lifecycle

### Phase 1: Development

**Branch**: `develop`

1. Create feature branch from `develop`:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/my-new-feature
   ```

2. Develop and test locally:
   ```bash
   make test
   make test-integration
   make lint
   ```

3. Commit and push feature branch:
   ```bash
   git add .
   git commit -m "feat: add my new feature"
   git push origin feature/my-new-feature
   ```

4. Create Pull Request to `develop`
5. After review and approval, merge to `develop`

---

### Phase 2: Staging Deployment

**Branch**: `staging`  
**Environment**: Staging Server  
**Port**: 8081

#### Step 1: Merge develop to staging

```bash
# Checkout staging branch
git checkout staging
git pull origin staging

# Merge from develop (no fast-forward to preserve history)
git merge develop --no-ff -m "chore: prepare release v1.2.0-rc1"

# Tag the release candidate
git tag v1.2.0-rc1
git push origin staging --tags
```

#### Step 2: Deploy to staging

```bash
# Option 1: Using Makefile
make deploy-staging

# Option 2: Using script directly
./scripts/deploy.sh staging v1.2.0-rc1
```

**What happens during deployment:**
1. Pre-deployment health check (if staging is running)
2. Database backup created
3. Docker image built with tag `v1.2.0-rc1`
4. Containers recreated with new image
5. Health checks validate deployment (max 60 seconds)
6. Smoke tests run automatically
7. Old Docker images cleaned up (keeps last 5)

#### Step 3: Validate staging deployment

```bash
# Check health
make health-check-staging

# Run integration tests
STAGING_URL=http://localhost:8081 make test-staging

# Check logs for errors
docker-compose -f docker-compose.staging.yml logs -f app
```

#### Step 4: Manual QA Testing

- [ ] Test all new features
- [ ] Test existing functionality (regression testing)
- [ ] Verify Discord bot connectivity
- [ ] Check for errors in logs
- [ ] Validate database migrations applied correctly

**If issues found:** Fix on `develop`, then repeat Phase 2 from Step 1.

---

### Phase 3: Production Deployment

**Branch**: `production`  
**Environment**: Production Server  
**Port**: 8080

> [!WARNING]
> Production deployment will cause **2-5 seconds of downtime**. Schedule deployments during low-traffic periods if possible.

#### Step 1: Merge staging to production

```bash
# Checkout production branch
git checkout production
git pull origin production

# Merge from staging (staging must be validated first!)
git merge staging --no-ff -m "chore: release v1.2.0"

# Tag the production release
git tag v1.2.0
git push origin production --tags
```

#### Step 2: Deploy to production

```bash
# Option 1: Using Makefile
make deploy-production

# Option 2: Using script directly
./scripts/deploy.sh production v1.2.0
```

**Deployment flow:**
1. **Confirmation prompt**: Type 'yes' to continue
2. **Pre-deployment health check**: Validates current production is healthy
3. **Database backup**: Creates timestamped backup file
4. **Build**: Docker image tagged `v1.2.0` and `latest-production`
5. **Deploy**: Containers recreated (database stays running)
6. **Health validation**: Waits up to 60 seconds for health checks
7. **Automatic rollback**: If health checks fail, previous version is restored
8. **Smoke tests**: Validates `/healthz` and `/progression/tree` endpoints

#### Step 3: Post-deployment validation

```bash
# Check health
make health-check-prod

# Check key endpoints
curl http://localhost:8080/healthz
curl http://localhost:8080/progression/tree

# Monitor logs for errors
docker-compose -f docker-compose.production.yml logs -f --tail=100 app

# Verify Discord bot is online and responsive
```

#### Step 4: Monitor production

- Check logs for errors over next 15-30 minutes
- Verify Discord bot responds to commands
- Monitor for user-reported issues

**If issues found:** Use rollback procedure immediately (see below).

---

## Rollback Procedures

### When to Rollback

Rollback immediately if:
- Health checks fail repeatedly
- Database migrations cause data corruption
- Critical bugs discovered in production
- Application crashes or becomes unresponsive
- Discord bot fails to connect

### Rollback Staging

```bash
# Option 1: Using Makefile
make rollback-staging

# Option 2: Using script directly (interactive)
./scripts/rollback.sh staging

# Option 3: Specify version directly
./scripts/rollback.sh staging v1.1.0
```

### Rollback Production

```bash
# Option 1: Using Makefile (interactive)
make rollback-production

# Option 2: Using script directly
./scripts/rollback.sh production v1.1.0
```

**Rollback process:**
1. Lists last 10 Docker images (if version not specified)
2. Prompts for target version to rollback to
3. **Confirmation prompt** (production only)
4. Stops current containers
5. Starts containers with previous image
6. Waits for health checks (60 seconds max)
7. **Optional**: Restore database backup

> [!CAUTION]
> **Database Rollback**  
> The rollback script will prompt to restore a database backup. Only do this if:
> - The failed deployment included destructive migrations
> - Data corruption occurred
> - You are certain the backup is from before the issue
> 
> Database restoration is **irreversible** and will lose any data created after the backup.

### Emergency Rollback (Manual)

If scripts fail, manual rollback:

```bash
# 1. List available images
docker images brandishbot

# 2. Set target version
export DOCKER_IMAGE_TAG=v1.1.0

# 3. Restart with previous image
docker-compose -f docker-compose.production.yml up -d --no-deps app discord

# 4. Verify health
curl http://localhost:8080/healthz
```

---

## Downtime Windows

### Expected Downtime

| Deployment Type | Downtime | Notes |
|----------------|----------|-------|
| Staging | ~2-5 seconds | No users affected |
| Production | ~2-5 seconds | Brief connection interruption |
| Database migrations (simple) | +1-2 seconds | Most migrations are fast |
| Database migrations (complex) | +5-30 seconds | Large data alterations |
| Rollback | ~2-5 seconds | Same as deployment |

### Minimizing Disruption

**During deployment:**
- Database is never stopped (stays running)
- Containers recreated without `down` command
- Health checks ensure app is ready before marking successful

**For users:**
- Discord bot: Brief disconnect (~2-5 sec)
- API clients: May see 1-2 failed requests during restart
- Active requests: Terminated (graceful shutdown not yet implemented)

### Future Improvements (Zero-Downtime)

For true zero-downtime deployment:
1. Add nginx load balancer with 2+ app instances
2. Implement graceful shutdown (drain connections before stop)
3. Use backward-compatible database migrations
4. Deploy in rolling fashion (one instance at a time)

**Out of scope for current implementation** as this is a single-user system.

---

## Hotfix Process

For critical bugs found in production that need immediate fixing:

### Option 1: Hotfix Branch (Recommended)

```bash
# 1. Create hotfix branch from production
git checkout production
git checkout -b hotfix/critical-bug-fix

# 2. Fix the bug and test locally
# ... make changes ...
make test
make test-integration

# 3. Commit the fix
git commit -am "fix: critical bug in X"

# 4. Merge to production immediately
git checkout production
git merge hotfix/critical-bug-fix --no-ff -m "hotfix: critical bug fix v1.2.1"
git tag v1.2.1

# 5. Deploy to production
make deploy-production

# 6. Backport fix to develop and staging
git checkout develop
git merge hotfix/critical-bug-fix
git push origin develop

git checkout staging
git merge develop
git push origin staging
```

### Option 2: Direct Production Fix (Emergency)

> [!CAUTION]
> Only use for true emergencies. Skips staging validation.

```bash
# 1. Fix on production branch
git checkout production
# ... make changes ...
git commit -am "fix: emergency fix"
git tag v1.2.1

# 2. Deploy immediately
make deploy-production

# 3. Backport to other branches ASAP
git checkout develop
git cherry-pick <commit-hash>
git push origin develop
```

---

## Versioning Scheme

Follow [Semantic Versioning](https://semver.org/):

- **Major** (v2.0.0): Breaking changes, incompatible API changes
- **Minor** (v1.2.0): New features, backwards-compatible
- **Patch** (v1.2.1): Bug fixes, backwards-compatible

**Release candidates**: `v1.2.0-rc1`, `v1.2.0-rc2`  
**Development tags**: `v1.2.0-dev` (auto-generated, not used for deployment)

---

## Quick Reference

### Common Commands

```bash
# Deploy to staging
make deploy-staging

# Deploy to production
make deploy-production

# Rollback production
make rollback-production

# Check health
make health-check-prod
make health-check-staging

# View logs
docker-compose -f docker-compose.production.yml logs -f app
docker-compose -f docker-compose.staging.yml logs -f app

# Run tests against staging
STAGING_URL=http://localhost:8081 make test-staging
```

### File Locations

- Deployment script: [scripts/deploy.sh](file:///home/osse1/projects/BrandishBot_Go/scripts/deploy.sh)
- Rollback script: [scripts/rollback.sh](file:///home/osse1/projects/BrandishBot_Go/scripts/rollback.sh)
- Health check: [scripts/health-check.sh](file:///home/osse1/projects/BrandishBot_Go/scripts/health-check.sh)
- Staging config: [docker-compose.staging.yml](file:///home/osse1/projects/BrandishBot_Go/docker-compose.staging.yml)
- Production config: [docker-compose.production.yml](file:///home/osse1/projects/BrandishBot_Go/docker-compose.production.yml)
- Database backups: `backup_<env>_<timestamp>.sql` (project root)

---

## Troubleshooting

### Deployment fails during build

**Symptom**: Docker build fails  
**Solution**: Check Docker logs, ensure `VERSION` arg is set, verify Dockerfile syntax

### Health checks timeout

**Symptom**: Deployment waits 60 seconds then fails  
**Solution**: 
- Check app logs: `docker-compose -f docker-compose.production.yml logs app`
- Verify database migrations completed
- Check for errors in entrypoint script

### Database migrations fail

**Symptom**: Entrypoint script reports migration errors  
**Solution**:
- Rollback to previous version
- Fix migration on `develop` branch
- Test migration on staging before re-deploying

### Rollback doesn't work

**Symptom**: Previous version also fails health checks  
**Solution**:
- Check if database schema is incompatible
- Restore database backup from before failed deployment
- Manual intervention may be required

---

## Additional Resources

- [Environment Setup Guide](file:///home/osse1/projects/BrandishBot_Go/docs/deployment/ENVIRONMENTS.md)
- [Staging Tests Documentation](file:///home/osse1/projects/BrandishBot_Go/docs/development/STAGING_TESTS.md)
- [Database Migrations Guide](file:///home/osse1/projects/BrandishBot_Go/docs/MIGRATIONS.md)
