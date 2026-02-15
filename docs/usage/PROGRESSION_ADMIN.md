# Progression System Admin Guide

## Table of Contents

1. [Overview](#overview)
2. [Admin Commands](#admin-commands)
3. [Common Tasks](#common-tasks)
4. [Monitoring](#monitoring)
5. [Troubleshooting](#troubleshooting)
6. [Best Practices](#best-practices)

---

## Overview

As an admin, you have full control over the progression system including:

- ✅ Force-unlocking features for testing or events
- 🔒 Relocking features if needed
- ⚡ Ending votes early with instant unlock
- 🔄 Resetting the entire tree (annual reset)
- 📊 Monitoring engagement and voting activity

**Base URL**: `http://localhost:8080/progression/admin`

---

## Admin Commands

### 1. Unlock a Feature/Item

**Use When**:

- Testing features before public unlock
- Special events or milestones
- Correcting issues from failed votes

**Command**:

```bash
curl -X POST http://localhost:8080/progression/admin/unlock \
  -H "Content-Type: application/json" \
  -d '{
    "node_key": "feature_buy",
    "level": 1
  }'
```

**Response**:

```json
{
  "message": "Node unlocked successfully",
  "node_key": "feature_buy",
  "level": 1
}
```

**⚠️ Important**: Check prerequisites first! Unlocking `feature_buy` without unlocking `feature_economy` and `item_money` will cause issues.

---

### 2. Relock a Feature/Item

**Use When**:

- Reverting accidental unlocks
- Temporarily disabling broken features
- Testing the locked state

**Command**:

```bash
curl -X POST http://localhost:8080/progression/admin/relock \
  -H "Content-Type: application/json" \
  -d '{
    "node_key": "feature_upgrade",
    "level": 1
  }'
```

**⚠️ Warning**: Relocking a node with dependent unlocked nodes will cause issues. Relock children first.

---

### 3. Instant Unlock (End Voting Early)

**Use When**:

- Voting has overwhelming support
- Special celebration/event
- Stuck voting that won't reach threshold

**Command**:

```bash
curl -X POST http://localhost:8080/progression/admin/instant-unlock \
  -H "Content-Type: application/json" \
  -d '{
    "node_key": "item_money"
  }'
```

**Effect**: Immediately ends active voting and unlocks the node.

---

### 4. Reset Progression Tree

**Use When**:

- Annual reset (recommended once per year)
- Major version update
- Starting fresh after testing

**Command**:

```bash
curl -X POST http://localhost:8080/progression/admin/reset \
  -H "Content-Type: application/json" \
  -d '{
    "reset_by": "admin_username",
    "reason": "Annual reset for 2026",
    "preserve_user_data": true
  }'
```

**Parameters**:

- `preserve_user_data: true` - Keeps user-specific unlocks (recipes)
- `preserve_user_data: false` - Wipes everything except root node

**⚠️ CRITICAL**: This is destructive! Backup database first:

```bash
pg_dump brandish_bot > backup_before_reset_$(date +%Y%m%d).sql
```

---

## Common Tasks

### Task 1: Unlock All Features for Testing

When setting up a test environment, unlock all features:

```bash
#!/bin/bash
# unlock_all_features.sh

API_URL="http://localhost:8080/progression/admin"

# Unlock in dependency order
features=(
  "item_money"
  "item_lootbox0"
  "feature_economy"
  "feature_buy"
  "feature_sell"
  "feature_upgrade"
  "feature_disassemble"
  "feature_search"
)

for feature in "${features[@]}"; do
  echo "Unlocking $feature..."
  curl -X POST "$API_URL/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"node_key\": \"$feature\", \"level\": 1}"
  echo ""
done

echo "All features unlocked!"
```

**Run**: `bash unlock_all_features.sh`

---

### Task 2: Check Current Unlock Status

View what's unlocked:

```bash
curl http://localhost:8080/progression/tree | \
  jq '.nodes[] | select(.is_unlocked == true) | {key: .node_key, level: .unlocked_level}'
```

**Output**:

```json
{"key": "progression_system", "level": 1}
{"key": "item_money", "level": 1}
{"key": "feature_economy", "level": 1}
```

---

### Task 3: Monitor Active Voting

Check if there's an active vote:

```bash
curl http://localhost:8080/progression/status | jq '.active_voting'
```

**Output**:

```json
{
  "node_key": "item_lootbox0",
  "display_name": "Basic Lootbox",
  "target_level": 1,
  "vote_count": 42,
  "votes_needed": 100,
  "ends_at": "2025-01-15T12:00:00Z"
}
```

---

### Task 4: Unlock Multi-Level Upgrade

For upgrades with multiple levels (e.g., cooldown reduction):

```bash
# Unlock level 1
curl -X POST http://localhost:8080/progression/admin/unlock \
  -d '{"node_key": "upgrade_cooldown_reduction", "level": 1}'

# Unlock level 2
curl -X POST http://localhost:8080/progression/admin/unlock \
  -d '{"node_key": "upgrade_cooldown_reduction", "level": 2}'

# ... up to level 5
```

---

### Task 5: View User Engagement

Check a user's contribution:

```bash
curl "http://localhost:8080/progression/engagement?user_id=user123" | jq
```

**Output**:

```json
{
  "user_id": "user123",
  "total_score": 245,
  "breakdown": {
    "messages_sent": 100,
    "commands_used": 30,
    "items_crafted": 15,
    "items_used": 20
  }
}
```

---

## Monitoring

### Key Metrics to Track

1. **Unlock Progress**: `total_unlocked / total_nodes`
2. **Active Engagement**: Daily/weekly active users contributing
3. **Vote Participation**: Unique voters per voting session
4. **Unlock Rate**: How often nodes are being unlocked

### SQL Queries for Monitoring

**Check unlock status**:

```sql
SELECT
  n.node_key,
  n.display_name,
  u.unlocked_at,
  u.unlocked_by
FROM progression_nodes n
LEFT JOIN progression_unlocks u ON n.id = u.node_id
ORDER BY u.unlocked_at DESC NULLS LAST;
```

**Top contributors**:

```sql
SELECT
  user_id,
  SUM(metric_value * w.weight) as total_score
FROM engagement_metrics em
JOIN engagement_weights w ON em.metric_type = w.metric_type
GROUP BY user_id
ORDER BY total_score DESC
LIMIT 10;
```

**Voting history**:

```sql
SELECT
  n.node_key,
  v.vote_count,
  v.voting_started_at,
  v.voting_ends_at,
  v.is_active
FROM progression_voting v
JOIN progression_nodes n ON v.node_id = n.id
ORDER BY v.voting_started_at DESC;
```

---

## Troubleshooting

### Issue 1: "Prerequisites not met" when voting

**Cause**: Parent nodes are not unlocked yet.

**Solution**:

1. Check the tree structure in [PROGRESSION_TREE.md](../planning/PROGRESSION_TREE.md)
2. Unlock parent nodes first (or use admin unlock)

**Example**: To unlock `feature_buy`:

```bash
# Must unlock in order:
curl -X POST .../unlock -d '{"node_key": "item_money", "level": 1}'
curl -X POST .../unlock -d '{"node_key": "feature_economy", "level": 1}'
curl -X POST .../unlock -d '{"node_key": "feature_buy", "level": 1}'
```

---

### Issue 2: Feature gate returns 403 even though unlocked

**Cause**: Cache issue or node_key mismatch.

**Solution**:

1. Verify the node is actually unlocked:
   ```bash
   curl http://localhost:8080/progression/tree | jq '.nodes[] | select(.node_key == "feature_buy")'
   ```
2. Check the feature key constant matches:
   ```go
   // Should be "feature_buy", not "buy" or "feature-buy"
   progression.FeatureEconomy == "feature_buy"
   ```
3. Restart the server to clear any caches

---

### Issue 3: Engagement not being tracked

**Cause**: Handler not recording engagement or async recording failing.

**Solution**:

1. Check server logs for "Failed to record engagement" errors
2. Verify database connection is healthy
3. Check `engagement_weights` table has entries:
   ```sql
   SELECT * FROM engagement_weights;
   ```
4. Test direct recording:
   ```go
   err := progressionService.RecordEngagement(ctx, "test_user", "message", 1)
   ```

---

### Issue 4: Voting stuck (won't end after 24 hours)

**Cause**: Clock skew or `voting_ends_at` not set properly.

**Solution**:

1. Check current time: `SELECT NOW();`
2. Check voting end time:
   ```sql
   SELECT voting_ends_at, is_active FROM progression_voting WHERE is_active = true;
   ```
3. End manually:
   ```bash
   curl -X POST .../admin/instant-unlock -d '{"node_key": "stuck_node"}'
   ```

---

## Best Practices

### 1. Testing Before Production

Always test admin commands in a staging environment first:

```bash
# Create a test database
createdb brandish_bot_test

# Run migrations
DATAAPI_URL=postgres://localhost/brandish_bot_test make migrate-up

# Test admin commands
curl -X POST http://localhost:8081/progression/admin/unlock ...

# Verify results
curl http://localhost:8081/progression/tree
```

---

### 2. Backup Before Major Operations

Before resets or bulk unlocks:

```bash
# Full backup
pg_dump brandish_bot > backup_$(date +%Y%m%d_%H%M%S).sql

# Just progression tables
pgdump -t 'progression_*' -t 'engagement_*' brandish_bot > progression_backup.sql
```

---

### 3. Announce Changes to Community

Before using admin commands that affect users:

1. **Announce in Discord**: "We're unlocking X feature early as a celebration!"
2. **Log the reason**: Include in the admin command (`reason` field)
3. **Document**: Keep a changelog of manual interventions

---

### 4. Monitor Engagement Weights

Adjust weights based on community behavior:

```sql
-- Lower message weight if spamming is an issue
UPDATE engagement_weights SET weight = 0.5 WHERE metric_type = 'message';

-- Increase crafting weight to encourage crafting
UPDATE engagement_weights SET weight = 5.0 WHERE metric_type = 'item_crafted';
```

**Restart required**: Server caches weights on startup.

---

### 5. Annual Reset Checklist

□ Announce reset 2 weeks in advance  
□ Backup database  
□ Verify `preserve_user_data` setting  
□ Run reset command  
□ Confirm only root node unlocked  
□ Check reset was logged:

```sql
SELECT * FROM progression_resets ORDER BY reset_at DESC LIMIT 1;
```

□ Announce completion  
□ Start community voting for first unlock

---

## Emergency Procedures

### Emergency: Accidental Full Reset

If you accidentally reset without `preserve_user_data`:

1. **Immediately restore from backup**:

   ```bash
   psql brandish_bot < backup_before_reset.sql
   ```

2. **If no backup exists**, contact users and offer compensation

3. **Prevent future**: Add confirmation prompts to reset scripts

---

### Emergency: Progression System Down

If progression endpoints return 500 errors:

1. Check database connection:

   ```bash
   psql brandish_bot -c "SELECT COUNT(*) FROM progression_nodes;"
   ```

2. Check migration status:

   ```bash
   psql brandish_bot -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 5;"
   ```

3. Restart server:

   ```bash
   systemctl restart brandish-bot
   # or
   pkill -SIGUSR2 main && ./main
   ```

4. Check logs:
   ```bash
   tail -f /var/log/brandish-bot/error.log | grep progression
   ```

---

## Support

**Questions?**

- Check [API Documentation](../api/PROGRESSION_API.md)
- View [Tree Structure](../planning/PROGRESSION_TREE.md)
- Review database schema in `migrations/0014_create_progression_tables.up.sql`

**Found a bug?**

- Create an issue with reproduction steps
- Include relevant logs and database state
- Mention which admin command was used

---

**Last Updated**: 2025-11-25  
**Version**: 1.0.0
