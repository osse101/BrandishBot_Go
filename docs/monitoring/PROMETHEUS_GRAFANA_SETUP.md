# Prometheus + Grafana Monitoring Setup

This guide covers the monitoring infrastructure for BrandishBot_Go, including Prometheus for metrics collection and Grafana for visualization.

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  API Server │────▶│  Prometheus  │────▶│   Grafana   │
│  :8080      │     │  :9090       │     │   :3000     │
│  /metrics   │     │  (scraper)   │     │ (dashboard) │
└─────────────┘     └──────────────┘     └─────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Alert Rules │
                    │  (optional)  │
                    └──────────────┘
```

**Components:**
- **API Server** (`app:8080/metrics`) - Exposes Prometheus metrics
- **Prometheus** (`prometheus:9090`) - Scrapes metrics, stores time-series data
- **Grafana** (`grafana:3000`) - Visualizes metrics with pre-built dashboards

**Metrics Exposed:**
- HTTP request metrics (rate, latency, errors)
- Business metrics (items sold/bought, money flow, crafting)
- Event system metrics (events published, handler errors)

---

## Quick Start

### Development Environment

1. **Start the monitoring stack:**
   ```bash
   make monitoring-up
   ```

2. **Access the dashboards:**
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (login: `admin` / `admin`)

3. **Generate test traffic:**
   ```bash
   make test
   # OR manually trigger API calls
   curl http://localhost:8080/healthz
   ```

4. **View metrics in Grafana:**
   - Navigate to Dashboards → BrandishBot - System Health
   - See request rate, latency percentiles, error rates

### Production Environment

1. **Configure secure credentials:**
   ```bash
   # Edit .env file
   GRAFANA_ADMIN_USER=admin
   GRAFANA_ADMIN_PASSWORD=your_strong_password_here
   GRAFANA_ROOT_URL=https://grafana.yourdomain.com
   ```

2. **Start production stack:**
   ```bash
   docker-compose -f docker-compose.production.yml up -d prometheus grafana
   ```

3. **Set up reverse proxy (Nginx/Caddy):**
   - **DO NOT expose ports 9090/3000 publicly**
   - Use reverse proxy with TLS termination
   - Example Nginx config:
     ```nginx
     location /grafana/ {
         proxy_pass http://localhost:3000/;
         proxy_set_header Host $host;
     }
     ```

4. **Configure authentication:**
   - Enable OAuth or LDAP in Grafana
   - See: https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/

---

## Environment-Specific Configurations

### Development (`docker-compose.yml`)
- **Ports:** Prometheus 9090, Grafana 3000 (exposed)
- **Scrape interval:** 15s
- **Retention:** 15 days
- **Credentials:** admin/admin (default)
- **Target:** `app:8080/metrics`

### Staging (`docker-compose.staging.yml`)
- **Ports:** Prometheus 9091, Grafana 3001 (avoid conflicts)
- **Scrape interval:** 20s
- **Retention:** 7 days
- **Target:** `app:8081/metrics` (staging API port)

### Production (`docker-compose.production.yml`)
- **Ports:** NOT exposed externally (security)
- **Scrape interval:** 30s
- **Retention:** 30 days
- **Resource limits:** Prometheus (2 CPU, 2GB RAM), Grafana (0.5 CPU, 512MB RAM)
- **Healthchecks:** Enabled with dependencies
- **Credentials:** Must be set in `.env` (strong password)

---

## Configuration Files

### Prometheus Configuration

**Base config:** `configs/prometheus/prometheus.yml`
```yaml
scrape_configs:
  - job_name: 'brandishbot-api'
    static_configs:
      - targets: ['app:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

**Production overrides:** `configs/prometheus/prometheus.production.yml`
- Increased scrape interval (30s)
- External labels for multi-cluster setups

**Staging overrides:** `configs/prometheus/prometheus.staging.yml`
- Target: `app:8081` (staging port)
- 20s scrape interval

### Alert Rules

**Location:** `configs/prometheus/alerts/brandishbot.rules.yml`

**Pre-configured alerts:**
- `HighHTTPErrorRate` - 5xx errors > 5% for 5 minutes
- `HighAPILatency` - p95 latency > 1s for 5 minutes
- `EventHandlerErrors` - Event errors > 0.1/sec for 5 minutes
- `HighInFlightRequests` - Concurrent requests > 100 for 5 minutes

**Alert severity levels:**
- `warning` - Requires attention, not urgent
- `critical` - Requires immediate action (future use)

### Grafana Provisioning

**Datasource:** `configs/grafana/provisioning/datasources/prometheus.yml`
- Auto-configures Prometheus as default datasource
- No manual setup required

**Dashboard provider:** `configs/grafana/provisioning/dashboards/default.yml`
- Loads dashboards from `configs/grafana/dashboards/`
- Dashboards appear automatically on first startup

---

## Available Dashboards

### 1. System Health
**File:** `configs/grafana/dashboards/system-health.json`

**Panels:**
- **Request Rate by Method** - GET/POST/PUT/DELETE req/sec
- **Latency Percentiles** - p50, p95, p99 response times
- **In-Flight Requests** - Current concurrent requests (gauge)
- **Error Rate by Status Code** - 2xx/4xx/5xx percentage
- **Top Endpoints by Volume** - Most-hit API paths (table)
- **Latency Heatmap** - Request duration distribution

**Use cases:**
- Monitor API health and performance
- Identify slow endpoints
- Track error rates

### 2. Business Metrics
**File:** `configs/grafana/dashboards/business-metrics.json`

**Panels:**
- **Item Trading Activity** - Items sold/bought rates
- **Top Items by Volume** - Most traded items (table)
- **Money Flow** - Money earned vs spent
- **Crafting Activity by Result** - Items crafted breakdown
- **Item Usage Trends** - Consumable item usage
- **Net Money Flow (1h)** - Economy balance (gauge)
- **Items Crafted (1h)** - Total crafting activity (gauge)

**Use cases:**
- Monitor economy health
- Identify popular items
- Track crafting trends

### 3. Events
**File:** `configs/grafana/dashboards/events.json`

**Panels:**
- **Events Published by Type** - Event breakdown (stacked)
- **Event Handler Errors by Type** - Error tracking
- **Event Type Distribution (1h)** - Pie chart of event types
- **Error Rate by Event Type** - Error percentage per type
- **Total Events (5m)** - Recent event volume (stat)
- **Total Errors (5m)** - Recent error count (stat)
- **Overall Error Rate** - Global error percentage (gauge)
- **Event Rate (1m)** - Current event throughput (stat)

**Use cases:**
- Monitor event bus health
- Identify problematic event handlers
- Track event throughput

---

## Makefile Commands

### Start/Stop
```bash
# Start monitoring stack
make monitoring-up

# Stop monitoring stack
make monitoring-down

# Restart services
make monitoring-restart
```

### Monitoring
```bash
# Check health status
make monitoring-status

# View logs (live)
make monitoring-logs
```

### Configuration Management
```bash
# Hot reload Prometheus config (no restart needed)
make prometheus-reload
```

---

## Troubleshooting

### Prometheus not scraping metrics

**Symptom:** Targets show as "DOWN" in Prometheus UI (http://localhost:9090/targets)

**Checks:**
1. Verify API server is running:
   ```bash
   curl http://localhost:8080/healthz
   ```

2. Check Prometheus can reach API:
   ```bash
   docker-compose exec prometheus wget -O- http://app:8080/metrics
   ```

3. Check Docker network:
   ```bash
   docker network inspect brandishbot_go_backend
   ```

4. Verify Prometheus config:
   ```bash
   docker-compose exec prometheus promtool check config /etc/prometheus/prometheus.yml
   ```

### Grafana dashboards not loading

**Symptom:** Dashboards don't appear or show "No data"

**Checks:**
1. Verify Prometheus datasource:
   - Grafana → Configuration → Data sources → Prometheus
   - URL should be `http://prometheus:9090`
   - Click "Test" button

2. Check dashboard provisioning:
   ```bash
   docker-compose exec grafana ls -la /etc/grafana/dashboards/
   ```

3. Verify Grafana logs:
   ```bash
   docker-compose logs grafana | grep -i error
   ```

4. Restart Grafana:
   ```bash
   docker-compose restart grafana
   ```

### High memory usage (Prometheus)

**Symptom:** Prometheus using > 2GB RAM

**Solutions:**
1. Reduce retention time:
   ```yaml
   # In docker-compose.yml
   command:
     - '--storage.tsdb.retention.time=7d'  # Reduce from 15d
   ```

2. Reduce scrape frequency:
   ```yaml
   # In prometheus.yml
   scrape_interval: 30s  # Increase from 15s
   ```

3. Add resource limits (production only):
   ```yaml
   # Already configured in docker-compose.production.yml
   deploy:
     resources:
       limits:
         memory: 2G
   ```

### Alert not firing

**Symptom:** Alert shows in Prometheus but doesn't trigger

**Checks:**
1. Check alert evaluation:
   - Navigate to http://localhost:9090/alerts
   - Verify alert is "Pending" or "Firing"

2. Test alert query manually:
   - Go to http://localhost:9090/graph
   - Run alert expression (e.g., `rate(http_requests_total{status=~"5.."}[5m])`)
   - Verify it returns data

3. Check `for` duration:
   - Alerts have a `for: 5m` duration
   - Condition must be true for 5 minutes before firing

### Grafana login issues

**Symptom:** Cannot login to Grafana

**Solutions:**
1. Reset admin password:
   ```bash
   docker-compose exec grafana grafana-cli admin reset-admin-password newpassword
   ```

2. Check environment variables:
   ```bash
   docker-compose exec grafana env | grep GRAFANA
   ```

3. Verify .env file has correct credentials:
   ```bash
   grep GRAFANA .env
   ```

---

## Performance Optimization

### Prometheus Query Optimization

**Slow queries:**
- Use `rate()` instead of `irate()` for smoother graphs
- Avoid high cardinality labels (e.g., user IDs)
- Use recording rules for complex queries

**Example recording rule:**
```yaml
# In configs/prometheus/alerts/brandishbot.rules.yml
groups:
  - name: recording_rules
    interval: 30s
    rules:
      - record: api:http_requests:rate5m
        expr: sum(rate(http_requests_total[5m])) by (method, path)
```

### Grafana Dashboard Performance

**Best practices:**
- Use time range selectors (1h, 6h, 24h)
- Limit table rows (`topk(10, ...)`)
- Use variables for dynamic filtering
- Enable query caching

---

## Security Best Practices

### Development
- ✅ Default credentials (admin/admin) acceptable
- ✅ Ports exposed on localhost only
- ⚠️ Use SSH tunnel for remote access

### Production
- ✅ Strong admin password in `.env`
- ✅ Ports NOT exposed externally
- ✅ Reverse proxy with TLS (Nginx/Caddy)
- ✅ Enable OAuth or LDAP authentication
- ✅ Restrict network access to `backend-production` network
- ⚠️ Regularly update Grafana/Prometheus images

---

## Backup and Restore

### Backup Prometheus Data
```bash
# Stop Prometheus
docker-compose stop prometheus

# Backup data volume
docker run --rm -v brandishbot_go_prometheus-data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz -C /data .

# Start Prometheus
docker-compose start prometheus
```

### Restore Prometheus Data
```bash
# Stop Prometheus
docker-compose stop prometheus

# Restore data volume
docker run --rm -v brandishbot_go_prometheus-data:/data -v $(pwd):/backup alpine tar xzf /backup/prometheus-backup.tar.gz -C /data

# Start Prometheus
docker-compose start prometheus
```

### Backup Grafana Dashboards
```bash
# Export all dashboards (requires API token)
curl -H "Authorization: Bearer YOUR_API_TOKEN" \
  http://localhost:3000/api/search?query=& | \
  jq -r '.[] | select(.type == "dash-db") | .uid' | \
  xargs -I {} curl -H "Authorization: Bearer YOUR_API_TOKEN" \
    http://localhost:3000/api/dashboards/uid/{} > dashboards-backup.json
```

---

## Advanced Configuration

### Alertmanager Integration (Future)

To add alerting notifications (Slack, email, etc.):

1. Add Alertmanager service to `docker-compose.yml`:
   ```yaml
   alertmanager:
     image: prom/alertmanager:latest
     ports:
       - "9093:9093"
     volumes:
       - ./configs/alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro
     networks:
       - backend
   ```

2. Configure Prometheus to use Alertmanager:
   ```yaml
   # In prometheus.yml
   alerting:
     alertmanagers:
       - static_configs:
           - targets: ['alertmanager:9093']
   ```

3. Configure notification channels in `alertmanager.yml`

### Custom Metrics

To add custom business metrics:

1. Update `internal/metrics/collector.go`:
   ```go
   // Add metric definition
   myCustomMetric := promauto.NewCounterVec(
       prometheus.CounterOpts{
           Name: "my_custom_metric_total",
           Help: "Description of metric",
       },
       []string{"label1", "label2"},
   )
   ```

2. Record metric in business logic:
   ```go
   myCustomMetric.WithLabelValues("value1", "value2").Inc()
   ```

3. Add to Grafana dashboard using PromQL

---

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)
- [BrandishBot Metrics Constants](../../internal/metrics/constants.go)
- [Dashboard Guide](./DASHBOARD_GUIDE.md)
