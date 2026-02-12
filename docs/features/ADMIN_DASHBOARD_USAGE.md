# Admin Dashboard — Usage Guide

This guide covers how to use the BrandishBot admin dashboard, including setup, configuration, and extensibility.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Configuration](#configuration)
3. [Accessing the Dashboard](#accessing-the-dashboard)
4. [Feature Guide](#feature-guide)
5. [Extensibility](#extensibility)
6. [Deployment Scenarios](#deployment-scenarios)
7. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Local Development

```bash
# 1. Start the backend API server
make docker-up          # Start PostgreSQL
make migrate-up         # Apply migrations
make build              # Build the application
./bin/app               # Run on port 8080 (or configured PORT)

# 2. Access the dashboard
# Open browser: http://localhost:8080/admin/

# 3. Login with your API key (from .env file)
API_KEY=your-secret-key-here
```

### Production Build

```bash
# Build frontend + Go binary in one step
make admin-build        # Build React app
make build              # Build Go binary with embedded frontend

# Run
./bin/app
```

---

## Configuration

### Backend Configuration

The admin dashboard uses the same configuration as the main BrandishBot API. Configuration is managed via environment variables (`.env` file or system environment).

#### Required Settings

```bash
# API Server
PORT=8080                    # Port the server listens on
API_KEY=your-secret-key      # API key for authentication

# Database (required for full functionality)
DB_HOST=localhost
DB_PORT=5432
DB_USER=brandishbot
DB_PASSWORD=your-db-password
DB_NAME=brandishbot
```

#### Optional Settings

```bash
# CORS & Security
TRUSTED_PROXIES=192.168.1.1  # Comma-separated list of trusted proxy IPs

# Development Mode
DEV_MODE=false               # Set to true to disable cooldowns
```

### Frontend Configuration

The frontend is built as a static SPA and embedded into the Go binary. Configuration happens at **build time**, not runtime.

#### Development Server (Vite)

File: `web/admin/vite.config.ts`

```typescript
export default defineConfig({
  plugins: [react()],
  base: '/admin/',           // Base path for assets
  server: {
    port: 5173,              // Dev server port
    proxy: {
      '/api': 'http://localhost:8080',      // Proxy API calls
      '/healthz': 'http://localhost:8080',
      '/readyz': 'http://localhost:8080',
      '/version': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080',
    },
  },
})
```

**When to modify:**
- ✅ Development — when backend runs on different port
- ❌ Production — not used (frontend served by Go)

#### Production Build

The production build is served directly by the Go server at `/admin/`, so **no frontend configuration is needed**. The frontend makes same-origin API calls.

---

## Accessing the Dashboard

### Connection Scenarios

#### Scenario 1: Same Machine (Localhost)

**Backend URL**: `http://localhost:8080`
**Dashboard URL**: `http://localhost:8080/admin/`

No configuration needed — this is the default.

#### Scenario 2: Remote Server (LAN or Cloud)

**Backend URL**: `http://192.168.1.100:8080`
**Dashboard URL**: `http://192.168.1.100:8080/admin/`

**Steps:**
1. Ensure backend server binds to `0.0.0.0` instead of `127.0.0.1`:
   ```bash
   # In .env or environment
   PORT=8080  # Go's http.Server uses ":8080" which binds to all interfaces
   ```

2. Configure firewall to allow port 8080:
   ```bash
   # Example: Ubuntu ufw
   sudo ufw allow 8080/tcp
   ```

3. Access from browser:
   ```
   http://<server-ip>:8080/admin/
   ```

4. Login with API key (same as in server's `.env`)

#### Scenario 3: Different Port

**Backend URL**: `http://localhost:9000`
**Dashboard URL**: `http://localhost:9000/admin/`

**Backend configuration:**
```bash
# .env
PORT=9000
```

No frontend changes needed — the dashboard is served at `http://localhost:9000/admin/` automatically.

#### Scenario 4: Behind Reverse Proxy (Nginx/Traefik)

**Public URL**: `https://brandishbot.example.com/admin/`
**Backend URL**: `http://localhost:8080` (internal)

**Nginx configuration:**
```nginx
server {
    listen 80;
    server_name brandishbot.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # SSE requires special handling
    location /api/v1/events {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
    }
}
```

**BrandishBot configuration:**
```bash
# .env
TRUSTED_PROXIES=<nginx-server-ip>  # For accurate client IP logging
```

No frontend changes needed.

#### Scenario 5: Docker Deployment

**Docker Compose:**
```yaml
version: '3.8'
services:
  brandishbot:
    image: brandishbot:latest
    ports:
      - "8080:8080"  # Map container port 8080 to host port 8080
    environment:
      - PORT=8080
      - API_KEY=${API_KEY}
      - DB_HOST=db
      # ... other env vars
    depends_on:
      - db

  db:
    image: postgres:15-alpine
    # ... db config
```

**Access:**
```
http://localhost:8080/admin/
```

To use a different host port:
```yaml
ports:
  - "9000:8080"  # Host port 9000 → container port 8080
```

Access: `http://localhost:9000/admin/`

---

## Feature Guide

### 1. Health Dashboard

**Path**: `/admin/`

**Features**:
- **Server Status**: Liveness (`/healthz`) and readiness (`/readyz`) checks
- **Build Info**: Version, Go version, build time, git commit
- **HTTP Metrics**: Request rate, latency (avg/p95), in-flight requests, error count
- **Event Metrics**: Events published, handler errors by type
- **Business Metrics**: Items sold/bought
- **SSE Status**: Number of connected SSE clients

**Refresh Rates**:
- Health checks: 10 seconds
- Metrics: 5 seconds
- Version info: 60 seconds

**Use Cases**:
- Monitor server health during deployments
- Track request latency and error rates
- Verify SSE connections are working

### 2. Admin Commands

**Path**: `/admin/commands`

**Tabs**:

#### Progression Tab
- **Unlock Node**: Unlock a specific progression node by index
- **Relock Node**: Relock a previously unlocked node
- **Start/Freeze/Force-End Voting**: Control voting sessions
- **Instant Unlock Leader**: Immediately unlock the current vote leader
- **Add Contribution**: Manually add engagement points to a user
- **Reset Tree**: ⚠️ Dangerous — Resets all progression data
- **Unlock All Nodes**: ⚠️ Dangerous — Unlocks all nodes at once

#### Jobs Tab
- **Award XP**: Give XP to a user for a specific job
- **Reset Daily XP**: ⚠️ Resets daily XP caps for all users

#### Cache Tab
- **View Cache Stats**: Display hit/miss rates and cache size
- **Reload Aliases**: Reload item name aliases from config
- **Reload Voting Weights**: Reload progression voting weights

#### Scenarios Tab
- **Run Scenario**: Execute predefined test scenarios
- **Custom Scenario**: Run custom JSON-defined scenarios

#### Timeouts Tab
- **Clear Timeout**: Remove timeout from a specific user

**Confirmation Dialogs**:
All destructive actions (reset tree, reset daily XP, unlock all) require confirmation.

**Error Handling**:
Errors are displayed as toast notifications. Successful actions show success toasts.

### 3. Live Events

**Path**: `/admin/events`

**Features**:
- **Real-time Event Stream**: SSE connection to `/api/v1/events`
- **Event Filtering**: Filter by category (Gamble, Expedition, Progression, Jobs, Timeout, Economy)
- **Auto-scroll**: Automatically scrolls to new events (pauses when user scrolls up)
- **Event Expansion**: Click any event to view full JSON payload
- **Clear Buffer**: Clear the event buffer (keeps connection alive)
- **Connection Status**: Shows connected/connecting/disconnected state

**Event Categories**:
- 🟡 **Gamble**: `gamble.*` events (amber)
- 🟢 **Expedition**: `expedition.*` events (emerald)
- 🟣 **Progression**: `progression.*` events (purple)
- 🔵 **Jobs**: `job.*` events (blue)
- 🔴 **Timeout**: `timeout.*` events (red)
- 🟡 **Economy**: `item.*`, `economy.*` events (yellow)

**Use Cases**:
- Monitor live game events during testing
- Debug event payloads in real-time
- Watch for specific event types (e.g., progression unlocks)

**Reconnection**:
If the SSE connection drops, it automatically reconnects with exponential backoff (1s → 2s → 4s → ... → max 30s).

### 4. User Management

**Path**: `/admin/users`

**Features**:

#### User Search
- Search by platform (Twitch/Discord/YouTube) + username
- Returns user profile with ID, platform links, creation date

#### Profile Tabs
- **Inventory**: View all items with quantities and quality levels
- **Jobs**: View all job levels and XP progress
- **Stats**: User statistics and event counts
- **Quests**: Active quests and completion progress
- **Events**: User's event history (last 50 events)

#### Admin Actions
- **Add Item**: Give items to the user
- **Remove Item**: Take items from the user
- **Award XP**: Give XP for any job
- **Clear Timeout**: Remove user timeout

**Use Cases**:
- Give items to users for compensation
- Debug user inventory issues
- Check quest progress
- View user's event history for debugging

---

## Extensibility

### Adding New Pages

#### 1. Create the Page Component

File: `web/admin/src/pages/NewPage.tsx`

```typescript
export function NewPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-gray-100">New Feature</h2>
      {/* Your content here */}
    </div>
  );
}
```

#### 2. Add Route

File: `web/admin/src/App.tsx`

```typescript
import { NewPage } from './pages/NewPage';

// In <Routes>:
<Route path="/admin/new-feature" element={<NewPage />} />
```

#### 3. Add Navigation Link

File: `web/admin/src/components/layout/Sidebar.tsx`

```typescript
const navItems = [
  // ... existing items
  { to: '/admin/new-feature', label: 'New Feature', icon: '🆕' },
];
```

#### 4. Rebuild

```bash
make admin-build
make build
```

### Adding New Backend Endpoints

#### 1. Create Handler

File: `internal/handler/admin_new_feature.go`

```go
package handler

import (
	"net/http"
)

type AdminNewFeatureHandler struct {
	// dependencies
}

func NewAdminNewFeatureHandler(/* deps */) *AdminNewFeatureHandler {
	return &AdminNewFeatureHandler{}
}

func (h *AdminNewFeatureHandler) HandleGetData(w http.ResponseWriter, r *http.Request) {
	// Your logic here
	respondJSON(w, http.StatusOK, map[string]string{"message": "Hello"})
}
```

#### 2. Register Route

File: `internal/server/server.go`

```go
// In the admin route section:
adminNewFeatureHandler := handler.NewAdminNewFeatureHandler(/* deps */)
r.Route("/admin", func(r chi.Router) {
	// ... existing routes
	r.Get("/new-feature/data", adminNewFeatureHandler.HandleGetData)
})
```

#### 3. Add Frontend API Call

File: `web/admin/src/pages/NewPage.tsx`

```typescript
import { useApi } from '../hooks/useApi';

export function NewPage() {
  const { data, error, isLoading } = useApi<{ message: string }>('/api/v1/admin/new-feature/data');

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p className="text-red-400">{error}</p>;

  return (
    <div>
      <h2>New Feature</h2>
      <p>{data?.message}</p>
    </div>
  );
}
```

#### 4. Rebuild

```bash
make build          # Backend
make admin-build    # Frontend (if API types changed)
make build          # Final build with new frontend
```

### Adding New Metrics

#### 1. Define Metric

File: `internal/metrics/metrics.go`

```go
var (
	NewMetric = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "new_metric_total",
			Help: "Description of new metric",
		},
	)
)
```

#### 2. Collect in Metrics Handler

File: `internal/handler/admin_metrics.go`

```go
// Add to AdminMetricsResponse:
type AdminMetricsResponse struct {
	// ... existing fields
	NewMetrics NewMetricsData `json:"new_metrics"`
}

type NewMetricsData struct {
	Total float64 `json:"total"`
}

// Add to gatherMetrics():
case "new_metric_total":
	for _, m := range mf.GetMetric() {
		resp.NewMetrics.Total += m.GetCounter().GetValue()
	}
```

#### 3. Update Frontend Type

File: `web/admin/src/api/types.ts`

```typescript
export interface AdminMetrics {
  // ... existing fields
  new_metrics: {
    total: number;
  };
}
```

#### 4. Display in Dashboard

File: `web/admin/src/pages/HealthPage.tsx`

```typescript
{metrics.data && (
  <MetricCard label="New Metric" value={formatNumber(metrics.data.new_metrics.total)} />
)}
```

### Adding Event Categories

File: `web/admin/src/pages/EventsPage.tsx`

```typescript
const EVENT_CATEGORIES: Record<string, { color: string; types: string[] }> = {
  // ... existing categories
  NewCategory: {
    color: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
    types: ['new_category.']
  },
};
```

Rebuild: `make admin-build && make build`

---

## Deployment Scenarios

### Single Server Deployment

```bash
# Server: example.com
# API runs on port 8080

# 1. Build
make admin-build
make build

# 2. Copy binary to server
scp bin/app user@example.com:/opt/brandishbot/

# 3. SSH and run
ssh user@example.com
cd /opt/brandishbot
PORT=8080 API_KEY=secret ./app

# 4. Access
http://example.com:8080/admin/
```

### Docker Deployment

```bash
# 1. Build Docker image (includes frontend build)
make docker-build

# 2. Run container
docker run -d \
  -p 8080:8080 \
  -e API_KEY=secret \
  -e DB_HOST=postgres \
  --name brandishbot \
  brandishbot:latest

# 3. Access
http://localhost:8080/admin/
```

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: Service
metadata:
  name: brandishbot
spec:
  ports:
    - port: 80
      targetPort: 8080
  selector:
    app: brandishbot
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: brandishbot
spec:
  replicas: 2
  selector:
    matchLabels:
      app: brandishbot
  template:
    metadata:
      labels:
        app: brandishbot
    spec:
      containers:
        - name: brandishbot
          image: brandishbot:latest
          ports:
            - containerPort: 8080
          env:
            - name: API_KEY
              valueFrom:
                secretKeyRef:
                  name: brandishbot-secrets
                  key: api-key
            - name: DB_HOST
              value: postgres-service
```

Access via Ingress or LoadBalancer at `/admin/`

### Multi-Instance with Load Balancer

**Setup**:
- Multiple BrandishBot instances behind load balancer
- Shared PostgreSQL database
- Sticky sessions NOT required (API is stateless)

**Load Balancer Config** (Nginx example):

```nginx
upstream brandishbot_backend {
    server 192.168.1.10:8080;
    server 192.168.1.11:8080;
    server 192.168.1.12:8080;
}

server {
    listen 80;
    server_name brandishbot.example.com;

    location / {
        proxy_pass http://brandishbot_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # SSE requires special handling
    location /api/v1/events {
        proxy_pass http://brandishbot_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
    }
}
```

**Note**: SSE connections will stick to one backend server. If that server goes down, the frontend will auto-reconnect to a different server.

---

## Troubleshooting

### Problem: "Invalid API key" error

**Cause**: API key in browser doesn't match server

**Solution**:
1. Check server's API key:
   ```bash
   grep API_KEY .env
   ```
2. Clear browser session storage:
   - Open DevTools → Application → Session Storage
   - Delete `brandishbot_api_key`
3. Refresh page and re-enter correct API key

### Problem: Dashboard shows "Not connected" (SSE)

**Cause**: SSE connection failed

**Solution**:
1. Check if backend is running:
   ```bash
   curl http://localhost:8080/healthz
   ```
2. Check if SSE endpoint works:
   ```bash
   curl -H "X-API-Key: your-key" http://localhost:8080/api/v1/events
   ```
3. Check browser console for errors
4. Verify API key is set in sessionStorage

### Problem: Metrics page shows empty data

**Cause**: No metrics have been collected yet

**Solution**:
1. Generate some traffic:
   ```bash
   curl http://localhost:8080/version
   curl http://localhost:8080/healthz
   ```
2. Check Prometheus endpoint:
   ```bash
   curl http://localhost:8080/metrics
   ```
3. Refresh dashboard

### Problem: Frontend changes don't appear

**Cause**: Old frontend embedded in Go binary

**Solution**:
```bash
make admin-build    # Rebuild frontend
make build          # Rebuild Go binary with new embedded assets
./bin/app           # Restart server
```

Hard refresh in browser: `Ctrl+Shift+R` (Windows/Linux) or `Cmd+Shift+R` (Mac)

### Problem: Can't access from remote machine

**Cause**: Server binding to localhost only, or firewall blocking

**Solution**:
1. Ensure server binds to all interfaces:
   ```bash
   # .env
   PORT=8080  # This binds to 0.0.0.0:8080 by default
   ```
2. Check firewall:
   ```bash
   sudo ufw allow 8080/tcp
   ```
3. Verify with:
   ```bash
   netstat -tulpn | grep 8080
   # Should show: 0.0.0.0:8080 or :::8080
   ```

### Problem: Docker build fails at frontend stage

**Cause**: Node version incompatibility or missing dependencies

**Solution**:
1. Check Docker build logs:
   ```bash
   docker build --no-cache -t brandishbot:debug .
   ```
2. Ensure `package-lock.json` exists:
   ```bash
   cd web/admin && npm install
   ```
3. Verify Node version in Dockerfile matches local dev:
   ```dockerfile
   FROM node:20-alpine AS frontend-builder
   ```

### Problem: Admin commands return 500 errors

**Cause**: Missing service dependencies or database connection

**Solution**:
1. Check server logs for error details
2. Verify database is running:
   ```bash
   make check-db
   ```
3. Check service initialization in `cmd/app/main.go`
4. Ensure all required services are passed to `server.NewServer()`

---

## Security Best Practices

### API Key Management

**DO**:
- ✅ Use a strong, random API key (32+ characters)
- ✅ Store in `.env` file (never commit to git)
- ✅ Rotate periodically
- ✅ Use different keys for dev/staging/production

**DON'T**:
- ❌ Use simple keys like "admin" or "password"
- ❌ Share keys in public channels
- ❌ Commit `.env` to version control

### Network Security

**Recommendations**:
- 🔒 Use HTTPS in production (reverse proxy with SSL/TLS)
- 🔒 Restrict admin dashboard to internal network or VPN
- 🔒 Use firewall rules to limit access
- 🔒 Enable rate limiting at reverse proxy level

### Session Security

The API key is stored in `sessionStorage` (not `localStorage`), which means:
- ✅ Cleared when browser tab closes
- ✅ Not shared across tabs
- ✅ Not persisted to disk
- ❌ Lost on page refresh (user must re-enter)

For longer sessions, consider implementing JWT tokens with refresh tokens.

---

## Advanced Configuration

### Custom Metrics Polling Interval

File: `web/admin/src/pages/HealthPage.tsx`

```typescript
// Change polling intervals:
const health = usePolling<HealthResponse>('/healthz', 10000);  // 10s
const metrics = usePolling<AdminMetrics>('/api/v1/admin/metrics', 5000);  // 5s
```

### Custom SSE Reconnect Strategy

File: `web/admin/src/hooks/useSSE.ts`

```typescript
// Adjust max delay and backoff multiplier:
const MAX_RECONNECT_DELAY = 30000;  // 30s max
const delay = Math.min(1000 * 2 ** retriesRef.current, MAX_RECONNECT_DELAY);
```

### Custom Event Buffer Size

File: `web/admin/src/hooks/useSSE.ts`

```typescript
const MAX_EVENTS = 500;  // Increase for more history
```

---

## Summary

### Quick Configuration Checklist

**When changing port**:
- [ ] Update `PORT` in `.env`
- [ ] Update URLs to `http://localhost:<new-port>/admin/`

**When deploying to different server**:
- [ ] Ensure server binds to `0.0.0.0` (default)
- [ ] Configure firewall to allow port
- [ ] Update `TRUSTED_PROXIES` if behind reverse proxy
- [ ] Access at `http://<server-ip>:<port>/admin/`

**When adding features**:
- [ ] Backend: Add handler + route in Go
- [ ] Frontend: Add page/component in React
- [ ] Rebuild: `make admin-build && make build`

**No configuration needed for**:
- ✅ Changing backend port (frontend uses same-origin)
- ✅ HTTPS (handled by reverse proxy)
- ✅ Load balancing (stateless API)
- ✅ Docker deployment (uses environment variables)

The admin dashboard is designed to require **zero frontend configuration** for most deployment scenarios — just configure the backend and rebuild!
