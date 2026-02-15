# Prometheus + Grafana Quick Start

## 5-Minute Setup (Development)

### Start Monitoring
```bash
make monitoring-up
```

**Access:**
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

### Generate Test Data
```bash
# Run tests to generate HTTP metrics
make test

# Or manually trigger API calls
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/user/inventory?username=testuser
```

### View Dashboards
1. Open Grafana: http://localhost:3000
2. Login with `admin` / `admin`
3. Navigate to Dashboards → Select:
   - **BrandishBot - System Health** (API performance)
   - **BrandishBot - Business Metrics** (economy)
   - **BrandishBot - Events** (event bus)

### Stop Monitoring
```bash
make monitoring-down
```

---

## Common Commands

```bash
# Start/Stop
make monitoring-up           # Start Prometheus + Grafana
make monitoring-down         # Stop monitoring stack
make monitoring-restart      # Restart services

# Monitoring
make monitoring-status       # Check health
make monitoring-logs         # View logs

# Configuration
make prometheus-reload       # Hot reload config
```

---

## Production Deployment Checklist

- [ ] Set strong Grafana password in `.env`:
  ```bash
  GRAFANA_ADMIN_PASSWORD=your_strong_password_here
  ```

- [ ] Start production stack:
  ```bash
  docker-compose -f docker-compose.production.yml up -d prometheus grafana
  ```

- [ ] Configure reverse proxy (Nginx/Caddy) with TLS
- [ ] Enable OAuth or LDAP authentication in Grafana
- [ ] Verify ports 9090/3000 NOT exposed externally
- [ ] Set up backup schedule for Prometheus data

---

## Troubleshooting

### Prometheus not scraping
```bash
# Check targets
curl http://localhost:9090/api/v1/targets | jq

# Verify config
docker-compose exec prometheus promtool check config /etc/prometheus/prometheus.yml
```

### Grafana dashboards missing
```bash
# Restart Grafana
docker-compose restart grafana

# Check logs
docker-compose logs grafana | grep -i error
```

### Need help?
See full guides:
- Setup: [PROMETHEUS_GRAFANA_SETUP.md](./PROMETHEUS_GRAFANA_SETUP.md)
- Dashboards: [DASHBOARD_GUIDE.md](./DASHBOARD_GUIDE.md)

---

## Key Metrics to Watch

### System Health
- **Request Rate** - Should be steady, spikes during events
- **Latency p95** - Keep < 200ms for good UX
- **Error Rate** - Should be < 1% (mostly 4xx client errors)

### Business
- **Money Flow** - Should balance over 24h
- **Item Trading** - Indicates player engagement
- **Crafting Activity** - Shows feature usage

### Events
- **Event Rate** - Steady flow indicates healthy system
- **Error Rate** - Should be near 0%
- **Handler Errors** - Investigate any sustained errors

---

## Dashboard URLs

**Quick Access:**
- System Health: http://localhost:3000/d/brandishbot-system-health
- Business Metrics: http://localhost:3000/d/brandishbot-business-metrics
- Events: http://localhost:3000/d/brandishbot-events

**Prometheus Targets:** http://localhost:9090/targets
**Prometheus Alerts:** http://localhost:9090/alerts

---

## Environment-Specific Ports

| Environment | Prometheus | Grafana | Notes |
|-------------|-----------|---------|-------|
| Development | 9090 | 3000 | Exposed on localhost |
| Staging | 9091 | 3001 | Alternate ports |
| Production | - | - | NOT exposed (reverse proxy only) |

---

## Next Steps

1. ✅ Start monitoring: `make monitoring-up`
2. ✅ Generate traffic: `make test`
3. ✅ View dashboards: http://localhost:3000
4. 📖 Read dashboard guide: [DASHBOARD_GUIDE.md](./DASHBOARD_GUIDE.md)
5. 🔔 Set up alerts: Configure Alertmanager (future)
6. 📊 Customize dashboards: Add panels for your metrics
