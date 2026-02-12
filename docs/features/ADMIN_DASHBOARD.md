# Admin Dashboard

The BrandishBot admin dashboard is an embedded React SPA providing GUI access to system health monitoring, admin commands, live event streaming, and user management.

## Overview

- **Frontend**: React 19 + TypeScript + Tailwind CSS + Vite
- **Backend**: Embedded via `//go:embed` with SPA routing fallback
- **Auth**: API key stored in `sessionStorage`, sent as `X-API-Key` header
- **Real-time**: SSE connection for live event streaming
- **URL**: `http://localhost:8080/admin/`

For detailed usage instructions, see the [Usage Guide](ADMIN_DASHBOARD_USAGE.md).

## Features

### 1. Health Dashboard (`/admin/`)
- Server liveness & readiness checks
- Build version information
- Prometheus metrics (HTTP, events, business, SSE)
- Real-time metric polling (5-10s intervals)

### 2. Admin Commands (`/admin/commands`)
5 tabbed panels:
- **Progression**: Unlock nodes, start/freeze voting, reset tree, add contributions
- **Jobs**: Award XP, reset daily XP caps
- **Cache**: View cache stats, reload aliases/weights
- **Scenarios**: Run admin test scenarios
- **Timeouts**: Clear user timeouts

### 3. Live Events (`/admin/events`)
- SSE-powered real-time event feed
- Event filtering by category (Gamble, Expedition, Progression, Jobs, Timeout, Economy)
- Auto-scroll with pause-on-scroll
- Event payload expansion (JSON viewer)
- 500-event ring buffer

### 4. User Management (`/admin/users`)
- User search by platform + username
- Tabbed profile view:
  - Inventory
  - Jobs & XP
  - Stats
  - Quests
  - Event history
- Admin actions:
  - Add/remove items
  - Award XP
  - Clear timeout

## Development

### Setup
```bash
# Install dependencies
make admin-install

# Run dev server (frontend only, proxies to localhost:8080)
make admin-dev

# Build for production
make admin-build
```

### Build Pipeline
1. `make admin-build` runs `npm ci && npm run build` in `web/admin/`
2. Copies `web/admin/dist/` to `internal/admin/dist/`
3. Go embeds `internal/admin/dist/` via `//go:embed all:dist`
4. `make build` compiles the Go binary with embedded frontend

### Docker
The `Dockerfile` includes a Node.js build stage that builds the frontend before the Go build stage:
```dockerfile
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
COPY web/admin/package.json web/admin/package-lock.json* ./
RUN npm ci
COPY web/admin/ .
RUN npm run build

FROM golang:1.24-alpine AS builder
# ... (copies frontend-builder dist to internal/admin/dist)
```

## Backend Endpoints

### New Endpoints
- `GET /api/v1/admin/metrics` — JSON metrics from Prometheus
- `GET /api/v1/admin/user/lookup?platform=X&username=Y` — User lookup
- `GET /api/v1/admin/events?user_id=X&event_type=Y&since=Z&limit=N` — Event log query

### Existing Endpoints (26+)
All existing admin endpoints are used by the dashboard:
- `/api/v1/progression/admin/*` (9 endpoints)
- `/api/v1/admin/jobs/*` (3 endpoints)
- `/api/v1/admin/cache/*` (1 endpoint)
- `/api/v1/admin/simulate/*` (4 endpoints)
- `/api/v1/admin/timeout/*` (1 endpoint)
- `/api/v1/admin/progression/*` (1 endpoint)
- Plus user/inventory/stats/quest endpoints

## Architecture

### Frontend Structure
```
web/admin/
├── src/
│   ├── api/
│   │   ├── client.ts          # API key injection, fetch wrapper
│   │   └── types.ts           # TypeScript types matching Go responses
│   ├── hooks/
│   │   ├── useAuth.ts         # Login/logout with sessionStorage
│   │   ├── useApi.ts          # Generic fetch with loading state
│   │   ├── usePolling.ts      # Interval-based data refresh
│   │   └── useSSE.ts          # SSE connection with reconnect
│   ├── components/
│   │   ├── layout/            # Sidebar, Header, Layout
│   │   ├── shared/            # StatusBadge, ConfirmDialog, DataTable, JsonViewer, Toast
│   │   └── commands/          # 5 command panel components
│   ├── pages/                 # LoginPage, HealthPage, CommandsPage, EventsPage, UsersPage
│   └── utils/                 # Formatters
├── index.html
├── package.json
└── vite.config.ts
```

### Backend Structure
```
internal/
├── admin/
│   ├── embed.go           # //go:embed all:dist
│   ├── handler.go         # SPA handler with fallback
│   └── handler_test.go    # Handler tests
└── handler/
    ├── admin_metrics.go   # Prometheus → JSON metrics
    ├── admin_user.go      # User lookup
    └── admin_events.go    # Event log query
```

## Authentication

The dashboard uses API key authentication:
1. User navigates to `/admin/` (public path, no auth required for HTML)
2. LoginPage prompts for API key
3. On submit, tests key with `GET /api/v1/admin/cache/stats`
4. If successful, stores key in `sessionStorage`
5. All subsequent API calls include `X-API-Key: <key>` header
6. Logout clears `sessionStorage`

## SSE Connection

The SSE hook uses `fetch` + `ReadableStream` instead of `EventSource` to support custom headers:
```typescript
const res = await fetch('/api/v1/events', {
  headers: { 'X-API-Key': apiKey },
  signal: controller.signal,
});

const reader = res.body.getReader();
// ... read stream line-by-line, parse SSE format
```

Exponential backoff reconnection: 1s → 2s → 4s → ... → max 30s

## Testing

```bash
# Backend tests
go test ./internal/admin/...
go test ./internal/handler/...

# Frontend type check
cd web/admin && npx tsc -b

# Frontend build
make admin-build

# Full integration
make build && ./bin/app
# Navigate to http://localhost:8080/admin/
```

## Common Tasks

### Add a New Admin Endpoint
1. Create handler in `internal/handler/admin_*.go`
2. Register route in `internal/server/server.go`
3. Add API call in `web/admin/src/api/client.ts` (if needed)
4. Use in a page component via `useApi` or `apiPost`

### Add a New Page
1. Create `web/admin/src/pages/NewPage.tsx`
2. Add route in `web/admin/src/App.tsx`
3. Add nav item in `web/admin/src/components/layout/Sidebar.tsx`

### Update Metrics
1. Add new metric in `internal/metrics/metrics.go`
2. Update `gatherMetrics()` in `internal/handler/admin_metrics.go`
3. Update `AdminMetricsResponse` type
4. Update frontend `AdminMetrics` type in `api/types.ts`
5. Display in `HealthPage.tsx`

## Troubleshooting

**Frontend doesn't update after code changes:**
```bash
make admin-build  # Rebuild frontend
make build        # Rebuild Go binary with new embedded assets
```

**API key not working:**
- Check that `/admin` is in `PublicPaths` (server/constants.go)
- Verify API key in browser DevTools → Application → Session Storage
- Check server logs for auth failures

**SSE not connecting:**
- Verify `/api/v1/events` endpoint is working (check server logs)
- Check browser DevTools → Network → EventSource or fetch requests
- Ensure API key is sent with SSE request (custom header support)

**Metrics empty:**
- Verify Prometheus metrics are being collected (`/metrics` endpoint)
- Check `prometheus.DefaultGatherer.Gather()` returns data
- Ensure metric names match in `gatherMetrics()` switch cases
