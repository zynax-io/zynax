# services/api-gateway — AGENTS.md

> **Language: Go 1.22+**
> Inherits all rules from root `AGENTS.md` and `services/AGENTS.md`.

---

## Purpose

The API Gateway is the **single external entry point** to the Keel
platform. It translates HTTP REST requests from external clients into gRPC
calls to internal services.

**Why Go:** `grpc-gateway` is the reference implementation for REST-to-gRPC
transcoding and is written in Go. `net/http` outperforms FastAPI for pure
HTTP routing at scale. Go's middleware composition model is idiomatic here.

**Responsibilities:**
- Expose a versioned REST API (`/api/v1/`).
- Authenticate and authorize external callers (API keys + JWT).
- Rate limit per client/API key using token bucket algorithm.
- Transcode REST ↔ gRPC via `grpc-gateway` annotations.
- Validate and sanitize all external inputs before forwarding.
- Log all requests for audit trail (separate from application logs).
- Return consistent error responses — never leak internal gRPC details.

**Non-responsibilities:**
- Does NOT implement business logic.
- Does NOT store data.
- Does NOT call backing services directly — only via gRPC.

---

## Internal Layout

```
services/api-gateway/
├── cmd/api-gateway/main.go
├── internal/
│   ├── api/
│   │   ├── router.go           ← http.ServeMux + grpc-gateway mux registration
│   │   ├── handlers/           ← Thin HTTP handlers for routes not covered by grpc-gateway
│   │   │   └── health.go
│   │   └── middleware/
│   │       ├── auth.go         ← JWT + API key validation
│   │       ├── ratelimit.go    ← Token bucket per client (Redis-backed)
│   │       ├── audit.go        ← Structured audit log for all mutating requests
│   │       ├── logging.go      ← Request/response structured logging
│   │       ├── recovery.go     ← Panic recovery → 500
│   │       └── cors.go         ← CORS headers
│   ├── domain/
│   │   └── auth.go             ← JWT claims validation, permission check logic
│   ├── infrastructure/
│   │   ├── clients.go          ← gRPC clients: registry, broker, memory
│   │   └── token_store.go      ← API key lookup in Redis
│   └── config/
│       └── config.go           ← prefix: KEEL_GW_
├── tests/
│   ├── features/api_gateway.feature
│   └── unit/
├── go.mod
└── Dockerfile
```

---

## grpc-gateway Pattern

```go
// cmd/api-gateway/main.go

// The gateway runs TWO servers:
//   1. gRPC server (for gRPC clients: grpc-gateway + direct gRPC consumers)
//   2. HTTP/1.1 server (REST clients via grpc-gateway mux)

grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

// grpc-gateway: registers HTTP-to-gRPC routes from proto annotations
gwMux := runtime.NewServeMux(
    runtime.WithErrorHandler(customErrorHandler),   // map gRPC status → HTTP status
    runtime.WithMetadata(propagateAuthMetadata),    // forward auth headers to upstream
)

// Register all upstream services
opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
pb.RegisterAgentRegistryServiceHandlerFromEndpoint(ctx, gwMux, cfg.RegistryURL, opts)
pb.RegisterTaskBrokerServiceHandlerFromEndpoint(ctx, gwMux, cfg.BrokerURL, opts)
pb.RegisterMemoryServiceHandlerFromEndpoint(ctx, gwMux, cfg.MemoryURL, opts)

// Compose middleware chain
handler := alice.New(
    middleware.Logging(logger),
    middleware.Recovery(),
    middleware.CORS(cfg.AllowedOrigins),
    middleware.Auth(tokenStore, jwtValidator),
    middleware.RateLimit(rateLimiter),
    middleware.Audit(auditLogger),
).Then(gwMux)

httpServer := &http.Server{
    Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
    Handler:      handler,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

---

## Auth Middleware

```go
// internal/api/middleware/auth.go

type Claims struct {
    jwt.RegisteredClaims
    Permissions []string `json:"permissions"`
    ClientID    string   `json:"client_id"`
}

func Auth(store TokenStore, validator *jwt.Parser) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip health/metrics endpoints
            if isPublicPath(r.URL.Path) { next.ServeHTTP(w, r); return }

            principal, err := extractPrincipal(r, store, validator)
            if err != nil {
                writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", err.Error())
                return
            }
            next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, principal)))
        })
    }
}

// requirePermission is a handler-level decorator (not middleware)
func requirePermission(perm string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            p := principalFrom(r.Context())
            if !p.HasPermission(perm) {
                writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "insufficient permissions")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Error Mapping (gRPC → HTTP)

```go
// internal/api/router.go

// customErrorHandler maps gRPC status codes to HTTP status + consistent JSON body.
// Never leaks internal error details to external callers.
func customErrorHandler(
    ctx context.Context, mux *runtime.ServeMux,
    marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error,
) {
    s, _ := status.FromError(err)
    httpCode := runtime.HTTPStatusFromCode(s.Code())

    // Sanitize: only expose details on client errors (4xx), not server errors (5xx)
    message := s.Message()
    if httpCode >= 500 { message = "internal error" }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpCode)
    json.NewEncoder(w).Encode(map[string]string{
        "code":    s.Code().String(),
        "message": message,
    })
}
```

---

## Rate Limiting (token bucket, Redis-backed)

```go
// internal/api/middleware/ratelimit.go

// Limits: 100 req/min per API key, 1000 req/hour per API key
// Headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset

func RateLimit(limiter *RedisRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := principalFrom(r.Context()).ClientID
            result, err := limiter.Allow(r.Context(), key)
            if err != nil {
                // Fail open — do not block requests when Redis is down
                slog.Warn("rate limiter unavailable", "err", err)
                next.ServeHTTP(w, r)
                return
            }
            w.Header().Set("X-RateLimit-Limit",     strconv.Itoa(result.Limit))
            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
            w.Header().Set("X-RateLimit-Reset",     strconv.FormatInt(result.ResetAt.Unix(), 10))
            if !result.Allowed {
                w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
                writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "rate limit exceeded")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## REST ↔ gRPC Route Map

| HTTP Method + Path | gRPC Service + Method | Notes |
|---|---|---|
| `POST /api/v1/agents` | `AgentRegistryService/RegisterAgent` | 201 on success |
| `GET /api/v1/agents/{id}` | `AgentRegistryService/GetAgent` | 404 on NOT_FOUND |
| `GET /api/v1/agents?capability=X` | `AgentRegistryService/ListAgentsByCapability` | Paginated |
| `DELETE /api/v1/agents/{id}` | `AgentRegistryService/DeregisterAgent` | 204 on success |
| `POST /api/v1/tasks` | `TaskBrokerService/SubmitTask` | 202 Accepted (async) |
| `GET /api/v1/tasks/{id}` | `TaskBrokerService/GetTask` | Poll for status |
| `DELETE /api/v1/tasks/{id}` | `TaskBrokerService/CancelTask` | 204 on success |
| `POST /api/v1/memory/namespaces` | `MemoryService/CreateNamespace` | 201 |
| `POST /api/v1/memory/{ns}/entries` | `MemoryService/SetEntry` | 200 |
| `GET /api/v1/memory/{ns}/search` | `MemoryService/SearchSimilar` | Vector search |

---

## Configuration

```go
// prefix: KEEL_GW_
type Config struct {
    HTTPPort         int    `envconfig:"HTTP_PORT"          default:"8080"`
    GRPCPort         int    `envconfig:"GRPC_PORT"          default:"9090"` // internal gRPC-web
    MetricsPort      int    `envconfig:"METRICS_PORT"       default:"9091"`
    RegistryURL      string `envconfig:"REGISTRY_URL"       required:"true"`
    BrokerURL        string `envconfig:"BROKER_URL"         required:"true"`
    MemoryURL        string `envconfig:"MEMORY_URL"         required:"true"`
    RedisURL         string `envconfig:"REDIS_URL"          required:"true"`
    JWTSecret        string `envconfig:"JWT_SECRET"         required:"true"`
    JWTIssuer        string `envconfig:"JWT_ISSUER"         default:"keel"`
    RateLimitPerMin  int    `envconfig:"RATE_LIMIT_PER_MIN" default:"100"`
    RateLimitPerHour int    `envconfig:"RATE_LIMIT_PER_HOUR" default:"1000"`
    AllowedOrigins   string `envconfig:"ALLOWED_ORIGINS"    default:"*"`
    ShutdownGraceSecs int   `envconfig:"SHUTDOWN_GRACE_SECS" default:"30"`
    LogLevel         string `envconfig:"LOG_LEVEL"          default:"INFO"`
    OtelEndpoint     string `envconfig:"OTEL_ENDPOINT"      default:"http://otel-collector:4317"`
    ServiceName      string `envconfig:"SERVICE_NAME"       default:"api-gateway"`
}
```

---

## BDD Scenarios

```gherkin
Feature: API Gateway

  Scenario: Register agent via REST returns 201 with agent_id
    Given a valid RegisterAgentRequest JSON body
    When POST /api/v1/agents is called with a valid API key
    Then the response status is 201
    And the response body contains a non-empty agent_id

  Scenario: Request without auth token returns 401
    When POST /api/v1/agents is called without an Authorization header
    Then the response status is 401
    And the response body contains code "UNAUTHENTICATED"

  Scenario: Request with insufficient permissions returns 403
    Given a token with permissions ["tasks:read"] only
    When POST /api/v1/agents is called (requires agents:write)
    Then the response status is 403

  Scenario: Rate limit exceeded returns 429 with Retry-After
    Given 101 requests have been made within 1 minute by the same client
    When the 102nd request is made
    Then the response status is 429
    And the Retry-After header is present

  Scenario: gRPC NOT_FOUND maps to HTTP 404
    When GET /api/v1/agents/non-existent-id is called
    Then the response status is 404

  Scenario: Internal gRPC errors return 500 without leaking details
    Given the agent-registry returns an INTERNAL gRPC error
    When GET /api/v1/agents/any-id is called
    Then the response status is 500
    And the message is exactly "internal error" (no stack trace)

  Scenario: Audit log entry created for every mutating request
    When POST /api/v1/tasks is called
    Then an audit log event is emitted with method, path, client_id, and timestamp
```
