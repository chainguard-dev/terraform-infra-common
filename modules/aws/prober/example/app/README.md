# Example Prober Application

This is a simple Go application that demonstrates how to build a prober for use with the AWS prober Terraform module.

## Features

- **Authorization**: Verifies the `Authorization` header matches the expected secret
- **Health Checks**: Performs configurable health checks against target URLs
- **JSON Response**: Returns detailed check results in JSON format
- **Logging**: Logs all check attempts and results

## Environment Variables

- `AUTHORIZATION` (required): Shared secret for authorization (set automatically by the module)
- `TARGET_URL` (optional): URL to check (default: `https://httpbin.org/status/200`)
- `PORT` (optional): Port to listen on (default: `8080`)

## Running Locally

```bash
# Set the authorization secret
export AUTHORIZATION="test-secret-123"

# Optional: Set target URL
export TARGET_URL="https://api.example.com"

# Run the application
go run main.go
```

## Testing Locally

```bash
# In another terminal, test with correct authorization
curl -H "Authorization: test-secret-123" http://localhost:8080/

# Test with incorrect authorization (should return 401)
curl -H "Authorization: wrong-secret" http://localhost:8080/

# Test the basic health endpoint (no auth required)
curl http://localhost:8080/health
```

## Expected Responses

### Successful Check (HTTP 200)

```json
{
  "status": "healthy",
  "timestamp": "2026-01-13T10:30:00Z",
  "checks": {
    "target_url": "OK",
    "dns": "OK",
    "memory": "OK"
  },
  "message": "All checks passed"
}
```

### Failed Check (HTTP 503)

```json
{
  "status": "unhealthy",
  "timestamp": "2026-01-13T10:30:00Z",
  "checks": {
    "target_url": "FAILED: request failed: connection timeout",
    "dns": "OK",
    "memory": "OK"
  },
  "message": "One or more checks failed"
}
```

### Unauthorized (HTTP 401)

```
Unauthorized
```

## Customizing the Prober

You can extend this prober to check various things:

### Database Connectivity

```go
import "database/sql"

func checkDatabase() error {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        return err
    }
    defer db.Close()

    return db.Ping()
}
```

### API Endpoint with Authentication

```go
func checkAuthenticatedAPI() error {
    client := &http.Client{Timeout: 10 * time.Second}

    req, err := http.NewRequest("GET", os.Getenv("API_URL"), nil)
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer " + os.Getenv("API_TOKEN"))

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("API returned status %d", resp.StatusCode)
    }

    return nil
}
```

### Redis/Cache Connectivity

```go
import "github.com/redis/go-redis/v9"

func checkRedis() error {
    rdb := redis.NewClient(&redis.Options{
        Addr: os.Getenv("REDIS_ADDR"),
    })

    ctx := context.Background()
    return rdb.Ping(ctx).Err()
}
```

## Deployment

When deployed via the Terraform module, this application will:

1. Be built into a container using `ko`
2. Be signed with `cosign`
3. Run on AWS App Runner
4. Receive the `AUTHORIZATION` environment variable automatically
5. Be checked by CloudWatch Synthetics every 5 minutes (configurable)

See the parent directory's `main.tf` for the complete deployment configuration.
