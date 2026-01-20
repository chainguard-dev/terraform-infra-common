# Example Go Application for AWS App Runner

A simple Go HTTP server that demonstrates AWS App Runner deployment with region-aware responses.

## Features

- **Health check endpoint** (`/health`) for App Runner health checks
- **Readiness endpoint** (`/ready`) for startup checks
- **JSON API** (`/`) with region information
- **Info endpoint** (`/info`) with detailed environment info
- **Web UI** (`/ui`) for browser testing
- **Environment-aware** - reads AWS region, environment, and custom variables

## Endpoints

### `GET /`
JSON response with service information:
```json
{
  "message": "Hello from AWS App Runner! 🚀",
  "region": "us-east-1",
  "region_name": "US East (N. Virginia)",
  "environment": "production",
  "hostname": "abc123def",
  "timestamp": "2025-01-09T10:00:00Z",
  "version": "1.0.0"
}
```

### `GET /health`
Health check endpoint (used by App Runner):
```json
{
  "status": "healthy",
  "region": "us-east-1",
  "uptime": "2h30m15s",
  "time": "2025-01-09T10:00:00Z"
}
```

### `GET /ready`
Readiness check endpoint:
```json
{
  "status": "ready",
  "region": "us-east-1"
}
```

### `GET /info`
Detailed information about the running service:
```json
{
  "version": "1.0.0",
  "region": "us-east-1",
  "region_name": "US East (N. Virginia)",
  "environment": "production",
  "hostname": "abc123def",
  "uptime": "2h30m15s",
  "started_at": "2025-01-09T07:30:00Z",
  "has_database_url": true,
  "has_api_key": true
}
```

### `GET /ui`
HTML page with a nice UI showing service information (great for browser testing).

## Environment Variables

The application reads these environment variables:

### Required
- `PORT` - Port to listen on (default: 8080)

### Optional
- `AWS_REGION` or `AWS_DEFAULT_REGION` - AWS region (auto-set by App Runner)
- `REGION_NAME` - Human-readable region name
- `ENVIRONMENT` - Environment name (e.g., production, staging)
- `LOG_LEVEL` - Logging level (default: info)
- `VERSION` - Application version (default: 1.0.0)

### Secrets (from AWS)
- `DATABASE_URL` - Database connection string
- `API_KEY` - External API key

## Local Development

### Run locally

```bash
cd app
go run main.go
```

The server will start on `http://localhost:8080`.

### Test endpoints

```bash
# Root endpoint
curl http://localhost:8080/

# Health check
curl http://localhost:8080/health

# Info endpoint
curl http://localhost:8080/info

# Web UI
open http://localhost:8080/ui
```

## Building

### Build locally

```bash
go build -o server main.go
./server
```

### Build with ko (for containers)

```bash
ko build --local .
```
