# Grafana Dashboard Guide

This guide explains how to interpret the BrandishBot Grafana dashboards and provides example queries for common use cases.

## Dashboard Overview

BrandishBot includes three pre-configured dashboards:
1. **System Health** - API performance and reliability
2. **Business Metrics** - Game economy and player activity
3. **Events** - Internal event system monitoring

All dashboards auto-refresh every 10 seconds and default to 1-hour time range.

---

## System Health Dashboard

**Purpose:** Monitor API server performance, identify bottlenecks, and track reliability metrics.

### Panel Descriptions

#### Request Rate by Method
**Metric:** `http_requests_total`
**Query:** `sum(rate(http_requests_total[5m])) by (method)`

**Interpretation:**
- Shows requests/second grouped by HTTP method (GET, POST, PUT, DELETE)
- Spike in POST requests may indicate high write activity
- Sudden drop to zero indicates API outage

**Normal behavior:**
- GET requests dominate (read-heavy API)
- POST requests steady during active gameplay
- PUT/DELETE requests lower volume

**Alert triggers:**
- Sudden 50%+ drop in request rate (possible outage)
- Spike 10x normal (possible attack/bot)

#### Latency Percentiles
**Metric:** `http_request_duration_seconds_bucket`
**Query:** `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`

**Interpretation:**
- **p50 (median):** 50% of requests faster than this
- **p95:** 95% of requests faster than this (SLA target)
- **p99:** 99% of requests faster than this (outliers)

**Normal behavior:**
- p50: < 50ms (fast endpoints)
- p95: < 200ms (acceptable)
- p99: < 1s (rare slow queries)

**Alert triggers:**
- p95 > 1s for 5 minutes (HighAPILatency alert)
- Gap between p95 and p99 growing (inconsistent performance)

**Example PromQL queries:**
```promql
# Average latency by endpoint
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))

# Top 5 slowest endpoints (p95)
topk(5, histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path)))
```

#### In-Flight Requests
**Metric:** `http_requests_in_flight`
**Query:** `http_requests_in_flight`

**Interpretation:**
- Current number of requests being processed simultaneously
- Gauge metric (instant value, not rate)

**Normal behavior:**
- 0-20 under normal load
- 20-50 during peak hours
- Spikes during gamble events (users joining)

**Alert triggers:**
- > 100 for 5 minutes (HighInFlightRequests alert)
- Indicates server struggling to process requests

#### Error Rate by Status Code
**Metric:** `http_requests_total{status=~"..."}`
**Query:** `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`

**Interpretation:**
- Percentage of requests returning 2xx/4xx/5xx status codes
- **2xx:** Success
- **4xx:** Client errors (bad requests, auth failures)
- **5xx:** Server errors (bugs, crashes, DB issues)

**Normal behavior:**
- 2xx: > 95%
- 4xx: < 5% (expected for invalid input)
- 5xx: < 1% (rare failures)

**Alert triggers:**
- 5xx > 5% for 5 minutes (HighHTTPErrorRate alert)

**Example PromQL queries:**
```promql
# Error rate by endpoint
sum(rate(http_requests_total{status=~"5.."}[5m])) by (path) / sum(rate(http_requests_total[5m])) by (path)

# Most common 4xx errors
topk(10, sum(rate(http_requests_total{status=~"4.."}[5m])) by (status))
```

#### Top Endpoints by Volume
**Metric:** `http_requests_total`
**Query:** `topk(10, sum(rate(http_requests_total[5m])) by (path))`

**Interpretation:**
- Table showing 10 most-hit API paths
- Helps identify hotspots and optimize frequently-used endpoints

**Normal behavior:**
- `/healthz` at top (health checks)
- `/api/v1/user/inventory` high (profile checks)
- `/api/v1/progression/status` high (voting queries)

**Optimization targets:**
- Endpoints with high req/sec + high latency → cache/optimize
- Unexpected high-volume endpoints → investigate abuse

#### Latency Heatmap
**Metric:** `http_request_duration_seconds_bucket`
**Query:** `sum(increase(http_request_duration_seconds_bucket[5m])) by (le)`

**Interpretation:**
- Heatmap showing distribution of request durations over time
- **X-axis:** Time
- **Y-axis:** Latency buckets (.001s to 10s)
- **Color:** Intensity (darker = more requests)

**Normal behavior:**
- Most requests concentrated in 0.01-0.1s range
- Few outliers in 1-10s range

**Patterns to watch:**
- Band moving upward → latency degrading over time
- Bimodal distribution → cache hits vs misses
- Vertical stripe → sudden spike (deployment, cache clear)

---

## Business Metrics Dashboard

**Purpose:** Monitor game economy health, identify popular features, and track player engagement.

### Panel Descriptions

#### Item Trading Activity
**Metrics:** `items_sold_total`, `items_bought_total`
**Query:** `sum(rate(items_sold_total[5m]))` / `sum(rate(items_bought_total[5m]))`

**Interpretation:**
- Two lines: items sold to economy, items bought from economy
- Ideal: roughly balanced (items cycling through system)
- Imbalance indicates money sink/faucet issues

**Normal behavior:**
- Sold rate slightly higher (players farming items)
- Both rates increase during events
- Flat overnight (low activity)

**Alert triggers:**
- Unusual spike (>100 items/sec for 15m) - possible exploit
- Sustained imbalance - economy needs rebalancing

**Example PromQL queries:**
```promql
# Net item flow (positive = more sold than bought)
sum(rate(items_sold_total[1h])) - sum(rate(items_bought_total[1h]))

# Item activity by rarity (requires rarity label)
sum(rate(items_sold_total[5m])) by (rarity)
```

#### Top Items by Volume
**Metric:** `items_sold_total`, `items_bought_total`
**Query:** `topk(10, sum(rate(items_sold_total[5m]) + rate(items_bought_total[5m])) by (item))`

**Interpretation:**
- Table showing most-traded items
- Helps identify:
  - Popular items (high engagement)
  - Potentially underpriced items (farmed heavily)
  - Bottleneck items (needed for recipes)

**Normal behavior:**
- Common items dominate (easy to obtain)
- Legendary items low volume (rare)
- Crafting materials high volume (consumed in recipes)

**Design insights:**
- High-volume items → good drop rates
- Zero-volume items → too rare or not useful
- Sudden spike → new recipe or meta shift

#### Money Flow
**Metrics:** `money_earned_total`, `money_spent_total`
**Query:** `sum(rate(money_earned_total[5m]))` / `sum(rate(money_spent_total[5m]))`

**Interpretation:**
- Two lines: money earned (selling items, rewards) vs spent (buying items)
- **Positive net flow:** Money entering economy (inflation risk)
- **Negative net flow:** Money leaving economy (deflation risk)

**Normal behavior:**
- Roughly balanced over 24h period
- Earned spikes during events (rewards)
- Spent spikes during sales (players buying)

**Alert triggers:**
- Money flow imbalance > 10k/sec for 30m (MoneyFlowImbalance alert)

**Example PromQL queries:**
```promql
# Net money flow over 1 hour
sum(increase(money_earned_total[1h])) - sum(increase(money_spent_total[1h]))

# Money sources breakdown (requires source label)
sum(rate(money_earned_total[5m])) by (source)
```

#### Crafting Activity by Result
**Metric:** `items_crafted_total`
**Query:** `sum(rate(items_crafted_total[5m])) by (result_item)`

**Interpretation:**
- Stacked area chart showing crafting activity
- Each color represents a different crafted item
- Height = total crafting rate

**Normal behavior:**
- Tier 1 items most common (low-level recipes)
- Legendary crafts rare (expensive recipes)
- Activity increases after XP events (new unlocks)

**Design insights:**
- Flat lines → recipe not used (too expensive?)
- Spikes → popular recipes (good balance)
- Missing items → recipe not discovered or too hard

**Example PromQL queries:**
```promql
# Crafting success rate (if tracked)
sum(rate(items_crafted_total{quality="masterwork"}[5m])) / sum(rate(items_crafted_total[5m]))

# Most popular recipes
topk(10, sum(rate(items_crafted_total[1h])) by (result_item))
```

#### Item Usage Trends
**Metric:** `items_used_total`
**Query:** `sum(rate(items_used_total[5m])) by (item)`

**Interpretation:**
- Consumable items being used (potions, buffs, keys)
- Separate line for each item type

**Normal behavior:**
- Steady usage of common consumables
- Spike usage during boss fights/events
- Rare item usage low but consistent

**Alert triggers:**
- Sudden drop to zero → bug in consumption system
- Unexpected spike → exploit or unintended mechanic

#### Net Money Flow (1h) Gauge
**Metric:** `money_earned_total`, `money_spent_total`
**Query:** `sum(increase(money_earned_total[1h])) - sum(increase(money_spent_total[1h]))`

**Interpretation:**
- Single number: net money change over last hour
- **Positive:** Money entering system (inflation)
- **Negative:** Money leaving system (deflation)

**Thresholds:**
- Green: -1000 to +1000 (balanced)
- Yellow: ±1000 to ±5000 (minor imbalance)
- Red: > ±5000 (major imbalance)

#### Items Crafted (1h) Gauge
**Metric:** `items_crafted_total`
**Query:** `sum(increase(items_crafted_total[1h]))`

**Interpretation:**
- Total items crafted in last hour
- Indicates crafting system engagement

**Normal behavior:**
- 100-500 during normal hours
- 500-2000 during events
- < 50 overnight

---

## Events Dashboard

**Purpose:** Monitor internal event system health, identify handler errors, and track event throughput.

### Panel Descriptions

#### Events Published by Type
**Metric:** `events_published_total`
**Query:** `sum(rate(events_published_total[5m])) by (type)`

**Interpretation:**
- Stacked area showing event breakdown by type
- Common types:
  - `engagement` - User engagement tracking
  - `gamble.started` - Gamble sessions
  - `gamble.complete` - Gamble results
  - `job.level_up` - Job progression
  - `progression.cycle_completed` - Voting cycles

**Normal behavior:**
- `engagement` dominates (high frequency)
- `gamble.complete` spikes during active sessions
- `progression.*` low but consistent

**Alert triggers:**
- Drop to zero → event bus failure
- Unexpected spike → event storm/retry loop

**Example PromQL queries:**
```promql
# Event rate by hour of day (requires timestamp aggregation)
sum(rate(events_published_total[1h])) by (type)

# Event type as percentage of total
sum(rate(events_published_total[5m])) by (type) / sum(rate(events_published_total[5m]))
```

#### Event Handler Errors by Type
**Metric:** `events_handler_errors_total`
**Query:** `sum(rate(events_handler_errors_total[5m])) by (type)`

**Interpretation:**
- Errors during event processing
- Separate line for each event type with errors

**Normal behavior:**
- Should be zero or near-zero
- Occasional blips acceptable (network transients)

**Alert triggers:**
- > 0.1 errors/sec for 5m (EventHandlerErrors alert)
- Sustained errors → handler bug or dependency failure

**Investigation steps:**
1. Check error type: `docker-compose logs app | grep ERROR`
2. Identify failing handler: Look for event type in logs
3. Check dependencies: Database, SSE hub, external APIs

#### Event Type Distribution (1h) Pie Chart
**Metric:** `events_published_total`
**Query:** `sum(increase(events_published_total[1h])) by (type)`

**Interpretation:**
- Pie chart showing proportion of each event type
- Visual representation of event mix

**Normal distribution:**
- Engagement: 60-70%
- Gamble events: 10-20%
- Job events: 5-10%
- Progression events: 5-10%
- Other: < 5%

**Anomalies:**
- Gamble > 50% → possible spam/exploit
- Engagement < 50% → low player activity

#### Error Rate by Event Type
**Metric:** `events_handler_errors_total`, `events_published_total`
**Query:** `sum(rate(events_handler_errors_total[5m])) by (type) / sum(rate(events_published_total[5m])) by (type)`

**Interpretation:**
- Percentage of events failing per type
- Helps identify which handlers are problematic

**Acceptable rates:**
- < 0.1% (1 in 1000) for all types
- Short-lived spikes acceptable

**Alert triggers:**
- > 1% for any type → investigate handler
- 100% for any type → handler completely broken

#### Total Events (5m) Stat
**Metric:** `events_published_total`
**Query:** `sum(increase(events_published_total[5m]))`

**Interpretation:**
- Total events in last 5 minutes
- Single large number display

**Normal behavior:**
- 1000-5000 during normal activity
- 5000-20000 during peak hours
- < 500 overnight

#### Total Errors (5m) Stat
**Metric:** `events_handler_errors_total`
**Query:** `sum(increase(events_handler_errors_total[5m]))`

**Interpretation:**
- Total errors in last 5 minutes
- **Goal:** Keep this at zero

**Alert thresholds:**
- Green: 0
- Yellow: 1-10 (transient issues)
- Red: > 10 (systemic problem)

#### Overall Error Rate Gauge
**Metric:** `events_handler_errors_total`, `events_published_total`
**Query:** `sum(rate(events_handler_errors_total[5m])) / sum(rate(events_published_total[5m]))`

**Interpretation:**
- Global event system error rate
- Gauge visualization with color-coded thresholds

**Thresholds:**
- Green: < 0.01 (< 1%)
- Yellow: 0.01-0.05 (1-5%)
- Red: > 0.05 (> 5%)

#### Event Rate (1m) Stat
**Metric:** `events_published_total`
**Query:** `sum(rate(events_published_total[1m]))`

**Interpretation:**
- Current event throughput (events/second)
- Real-time system load indicator

**Normal behavior:**
- 5-20 events/sec during normal hours
- 20-100 events/sec during peak
- < 5 events/sec overnight

---

## Common PromQL Query Patterns

### Filtering by Labels
```promql
# Specific HTTP method
http_requests_total{method="POST"}

# Specific endpoint
http_requests_total{path="/api/v1/gamble/start"}

# Multiple statuses
http_requests_total{status=~"5.."}  # All 5xx errors
```

### Aggregation Functions
```promql
# Sum across all labels
sum(http_requests_total)

# Sum grouped by method
sum(http_requests_total) by (method)

# Average latency
avg(http_request_duration_seconds)

# Top 10 endpoints
topk(10, sum(rate(http_requests_total[5m])) by (path))
```

### Rate and Increase
```promql
# Per-second rate over 5 minutes
rate(http_requests_total[5m])

# Total count over 1 hour
increase(http_requests_total[1h])

# Rate of change (derivative)
deriv(http_requests_total[5m])
```

### Histogram Percentiles
```promql
# 95th percentile latency
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 99.9th percentile (tail latency)
histogram_quantile(0.999, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
```

### Combining Metrics
```promql
# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# Net money flow
sum(rate(money_earned_total[5m])) - sum(rate(money_spent_total[5m]))

# Success rate
sum(rate(http_requests_total{status=~"2.."}[5m])) / sum(rate(http_requests_total[5m]))
```

---

## Alert Interpretation

### HighHTTPErrorRate
**Trigger:** 5xx error rate > 5% for 5 minutes

**Possible causes:**
- Database connection pool exhausted
- Panic in handler code
- External dependency failure (Streamer.bot, Discord)

**Investigation steps:**
1. Check logs: `docker-compose logs app | grep ERROR`
2. Check DB connections: `SELECT count(*) FROM pg_stat_activity;`
3. Test endpoints manually: `curl http://localhost:8080/api/v1/healthz`

### HighAPILatency
**Trigger:** p95 latency > 1s for 5 minutes

**Possible causes:**
- Slow database query (missing index)
- High CPU usage (complex computation)
- Network latency (external API call)

**Investigation steps:**
1. Identify slow endpoint: Check "Top Endpoints by Volume" + latency
2. Check database: `SELECT * FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;`
3. Profile code: Use pprof (`curl http://localhost:8080/debug/pprof/profile`)

### EventHandlerErrors
**Trigger:** Event handler errors > 0.1/sec for 5 minutes

**Possible causes:**
- SSE hub disconnected
- Database write failure
- Handler panic

**Investigation steps:**
1. Check event logs: `docker-compose logs app | grep "event handler"`
2. Identify failing event type: Check "Event Handler Errors by Type" panel
3. Check handler code: `internal/*/handler.go`

### HighInFlightRequests
**Trigger:** Concurrent requests > 100 for 5 minutes

**Possible causes:**
- Slow endpoint blocking workers
- Gamble session with many participants
- DDoS attack

**Investigation steps:**
1. Check active requests: `curl http://localhost:8080/metrics | grep in_flight`
2. Identify endpoint: Check logs for long-running requests
3. Scale resources: Increase container CPU/memory limits

---

## Best Practices

### Dashboard Usage
1. **Set appropriate time range** - 1h for real-time, 24h for trends, 7d for patterns
2. **Use refresh interval** - 10s for monitoring, 30s for analysis
3. **Add annotations** - Mark deployments, incidents, config changes
4. **Create alerts** - Don't rely on manual checking

### Query Optimization
1. **Use recording rules** for complex queries
2. **Limit cardinality** - Avoid high-cardinality labels (user IDs)
3. **Use rate() over irate()** - Smoother graphs
4. **Cache dashboard queries** - Enable Grafana query caching

### Investigation Workflow
1. **Identify anomaly** - Dashboard shows unusual pattern
2. **Narrow time range** - Zoom to exact time of issue
3. **Check correlated metrics** - Look at related panels
4. **Drill down** - Run custom queries to isolate root cause
5. **Check logs** - Correlate metrics with application logs
6. **Document findings** - Add annotation to dashboard

---

## Next Steps

- **Customize dashboards** - Add panels for your specific use cases
- **Create alerts** - Set up Alertmanager for notifications
- **Add more metrics** - Instrument new code with Prometheus
- **Explore advanced features** - Recording rules, federation, long-term storage

## References

- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)
- [Grafana Query Editor](https://grafana.com/docs/grafana/latest/panels/query-a-data-source/)
- [BrandishBot Metrics](../../internal/metrics/metrics.go)
- [Setup Guide](./PROMETHEUS_GRAFANA_SETUP.md)
