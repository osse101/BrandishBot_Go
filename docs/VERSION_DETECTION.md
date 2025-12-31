# Version Detection Guide

## Quick Check: Is My Deployment Out of Sync?

Simply call the `/version` endpoint to see what version is currently running:

```bash
# Check local version
curl http://localhost:8080/version

# Check staging version  
curl http://your-staging-server:8080/version

# Check production version
curl http://your-prod-server:8080/version
```

**Example Response:**
```json
{
  "version": "53cf434-dirty",
  "go_version": "go1.24",
  "build_time": "2025-12-31_19:45:23",
  "git_commit": "53cf434"
}
```

## Understanding the Version Fields

- **version**: Git tag or commit hash (e.g., `v1.2.3` or `53cf434`)
  - Includes `-dirty` if there are uncommitted changes
- **go_version**: Go compiler version used to build the binary
- **build_time**: UTC timestamp when the binary was built  
- **git_commit**: Short git commit hash (first 7 characters)

## How It Works

### Build-Time Injection

Version information is injected at build time using Go's `-ldflags` flag:

1. **Local Builds** (`make build`):
   - Runs `git describe --tags --always --dirty` to get version
   - Injects current timestamp and git commit  
   - Embeds into `internal/handler.Version`, `BuildTime`, `GitCommit`

2. **Docker Builds** (`make docker-build`):
   - Passes `VERSION`, `BUILD_TIME`, `GIT_COMMIT` as build arguments
   - Docker embeds these via `-ldflags` during image build
   - Values persist in the container image

### Deployment Workflow

When you run:
```bash
git pull
make docker-build
make deploy-staging
```

The build process:
1. Gets current git info (`53cf434` from `git describe`)
2. Builds image with that version embedded
3. Deploys container with version baked in

### Checking for Desyncs

**Scenario**: You commit changes but staging still shows old errors.

**Check**:
```bash
# What's your local HEAD?
git log -1 --oneline
# Output: 53cf434 Fix AI error

# What's deployed?
curl http://staging:8080/version
# Output: {"version":"3d8e7b7",...}
```

**Result**: Staging is running an older commit (`3d8e7b7`). You need to rebuild and redeploy.

## Common Scenarios

### Scenario 1: Deployed Code is Old
```bash
$ curl http://staging:8080/version  
{"version":"3d8e7b7","build_time":"2025-12-30_10:15:32"}

$ git log -1 --oneline
53cf434 Fix search feature
```

**Solution**: Redeploy
```bash
cd /path/to/staging
git pull
make docker-build
make deploy-staging
```

### Scenario 2: Local Changes Not Committed
```bash
$ curl http://localhost:8080/version
{"version":"53cf434-dirty",...}
```

The `-dirty` suffix means you have uncommitted changes. Commit them before deploying.

### Scenario 3: Verifying Successful Deployment
```bash
# Before deploy
curl http://staging:8080/version  # v1.2.0

# After deploy
make deploy-staging
curl http://staging:8080/version  # v1.2.1 ✓
```

##Automation Tips

### Add to Deployment Script
```bash
#!/bin/bash
echo "Current deployed version:"
curl -s http://staging:8080/version | jq .

echo "Deploying..."
make deploy-staging

sleep 5
echo "New deployed version:"
curl -s http://staging:8080/version | jq .
```

### Pre-Deployment Check
```bash
# Add to your workflow
LOCAL_COMMIT=$(git rev-parse --short HEAD)
DEPLOYED_COMMIT=$(curl -s http://staging:8080/version | jq -r .git_commit)

if [ "$LOCAL_COMMIT" == "$DEPLOYED_COMMIT" ]; then
  echo "✓ Staging is up to date"
else
  echo "⚠ Staging is behind (deployed: $DEPLOYED_COMMIT, local: $LOCAL_COMMIT)"
fi
```

### Health Check Dashboard
Create a simple monitoring dashboard that polls `/version` and compares with your git repository to detect drift.

## Technical Details

### Version Handler Code
Located in `/internal/handler/version.go`:
```go
var (
    Version   = "dev"          // Set via -X flag
    BuildTime = "unknown"      
    GitCommit = "unset"        
)
```

### Build Command (Makefile)
```makefile
VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

go build -ldflags "-X github.com/osse101/BrandishBot_Go/internal/handler.Version=$$VERSION ..."
```

### Docker Build (Dockerfile)
```dockerfile
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN go build -ldflags="-X ...Version=${VERSION} -X ...BuildTime=${BUILD_TIME} ..."
```

## Troubleshooting

### Version shows "dev"
- Not built with `make build` or `make docker-build`
- Built directly with `go build` (bypasses version injection)
- **Fix**: Use `make build` instead

### Version shows old commit despite rebuilding
- Docker cached old image layer
- **Fix**: Use `make docker-build` (includes `--no-cache`)

### Build time shows "unknown"
- System doesn't have `date` command or incorrect format
- **Fix**: Ensure `date` is available in build environment
