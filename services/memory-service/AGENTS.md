# services/memory-service — AGENTS.md

> **Language: Go 1.22+**
> Inherits all rules from root `AGENTS.md` and `services/AGENTS.md`.

---

## Purpose

The Memory Service provides **namespaced, shared memory** for agents.
It is the only persistent context store in the mesh.

**Two storage types:**

| Type | Backend | Use Case |
|------|---------|---------|
| Key-Value | Redis | Fast ephemeral state, agent scratch-pads, session data |
| Vector | PostgreSQL + pgvector | Semantic similarity search, RAG, long-term memory |

**Responsibilities:**
- Create and delete namespaces with optional TTL.
- Set, get, delete, and list key-value entries per namespace.
- Upsert and delete vector embeddings per namespace.
- Semantic similarity search (ANN via pgvector).
- Namespace isolation (agents cannot read each other's namespaces without permission).
- Audit log of all write operations.

**Non-responsibilities:** Does not generate embeddings (caller provides the vector).
Does not store task results (use task-broker for that).

---

## Internal Layout

```
services/memory-service/
├── cmd/memory-service/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← All 10 RPCs mapped to domain
│   ├── domain/
│   │   ├── model.go            ← NamespaceID, MemoryKey, MemoryEntry, VectorEntry, SimilarityResult
│   │   ├── service.go          ← KVService, VectorService
│   │   ├── repository.go       ← KVRepository, VectorRepository interfaces
│   │   └── errors.go           ← ErrNamespaceNotFound, ErrKeyNotFound, ErrDimensionMismatch
│   ├── infrastructure/
│   │   ├── redis_kv.go         ← RedisKVRepository
│   │   ├── postgres_vector.go  ← PostgresVectorRepository (pgvector)
│   │   └── namespace_store.go  ← Namespace metadata in PostgreSQL
│   └── config/
│       └── config.go           ← prefix: KEEL_MEMORY_
├── tests/
│   ├── features/memory_service.feature
│   └── unit/
├── go.mod
└── Dockerfile
```

---

## Domain Model

```go
// internal/domain/model.go

// NamespaceID format: <type>:<identifier>
// Types: agent, task, session, global
// Examples: "agent:analyst-01", "task:t-abc123", "global"
type NamespaceID string

func (n NamespaceID) Validate() error {
    parts := strings.SplitN(string(n), ":", 2)
    if len(parts) != 2 { return fmt.Errorf("%w: %q missing colon separator", ErrInvalidNamespace, n) }
    validTypes := map[string]bool{"agent": true, "task": true, "session": true, "global": true}
    if !validTypes[parts[0]] { return fmt.Errorf("%w: unknown type %q", ErrInvalidNamespace, parts[0]) }
    return nil
}

type MemoryKey string

type Namespace struct {
    ID               NamespaceID
    VectorDimensions int        // 0 = KV-only namespace. Fixed at creation.
    CreatedAt        time.Time
    ExpiresAt        *time.Time
}

type MemoryEntry struct {
    Key       MemoryKey
    Value     []byte
    ExpiresAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}

type VectorEntry struct {
    ID          string
    Embedding   []float32
    Metadata    map[string]any
    ContentHash string    // SHA-256 of original content — for dedup
    CreatedAt   time.Time
}

type SimilarityResult struct {
    Entry VectorEntry
    Score float32 // cosine similarity: 0.0–1.0
    Rank  int     // 1-indexed
}

const (
    DefaultTopK          = 10
    MaxTopK              = 100
    DefaultMinSimilarity = float32(0.7)
)
```

---

## Domain Services

```go
// internal/domain/service.go

type KVRepository interface {
    Set(ctx context.Context, ns NamespaceID, entry MemoryEntry) error
    Get(ctx context.Context, ns NamespaceID, key MemoryKey) (*MemoryEntry, error)
    Delete(ctx context.Context, ns NamespaceID, key MemoryKey) error
    List(ctx context.Context, ns NamespaceID, prefix MemoryKey, page Page) ([]MemoryEntry, string, error)
    DeleteNamespace(ctx context.Context, ns NamespaceID) error
}

type VectorRepository interface {
    Upsert(ctx context.Context, ns NamespaceID, entry VectorEntry) error
    Search(ctx context.Context, ns NamespaceID, query []float32, topK int, minScore float32) ([]SimilarityResult, error)
    Delete(ctx context.Context, ns NamespaceID, id string) error
    DeleteNamespace(ctx context.Context, ns NamespaceID) error
}

type NamespaceRepository interface {
    Create(ctx context.Context, ns Namespace) error
    Get(ctx context.Context, id NamespaceID) (*Namespace, error)
    Delete(ctx context.Context, id NamespaceID) error
}

type KVService struct {
    namespaces NamespaceRepository
    kv         KVRepository
}

func (s *KVService) Set(ctx context.Context, nsID NamespaceID, key MemoryKey, value []byte, ttl *time.Duration) error {
    if _, err := s.namespaces.Get(ctx, nsID); err != nil {
        return fmt.Errorf("namespace %q: %w", nsID, ErrNamespaceNotFound)
    }
    entry := MemoryEntry{Key: key, Value: value, UpdatedAt: time.Now().UTC()}
    if ttl != nil {
        exp := time.Now().UTC().Add(*ttl)
        entry.ExpiresAt = &exp
    }
    return s.kv.Set(ctx, nsID, entry)
}

type VectorService struct {
    namespaces NamespaceRepository
    vectors    VectorRepository
}

func (s *VectorService) Search(
    ctx context.Context, nsID NamespaceID,
    query []float32, topK int, minScore float32,
) ([]SimilarityResult, error) {
    ns, err := s.namespaces.Get(ctx, nsID)
    if err != nil { return nil, fmt.Errorf("%w: %s", ErrNamespaceNotFound, nsID) }
    if ns.VectorDimensions == 0 {
        return nil, fmt.Errorf("%w: namespace %q has no vector dimensions", ErrKVOnlyNamespace, nsID)
    }
    if len(query) != ns.VectorDimensions {
        return nil, fmt.Errorf("%w: got %d, expected %d", ErrDimensionMismatch, len(query), ns.VectorDimensions)
    }
    k := min(topK, MaxTopK)
    return s.vectors.Search(ctx, nsID, query, k, minScore)
}
```

---

## pgvector Index (SQL)

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE namespaces (
    id                 TEXT PRIMARY KEY,
    vector_dimensions  INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at         TIMESTAMPTZ
);

CREATE TABLE vector_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id TEXT NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    embedding    vector(1536),   -- dimension set per namespace
    metadata     JSONB NOT NULL DEFAULT '{}',
    content_hash TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- IVFFlat ANN index — rebuild when data grows significantly
CREATE INDEX idx_vector_embedding
    ON vector_entries USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

CREATE INDEX idx_vector_namespace ON vector_entries(namespace_id);
```

---

## Configuration

```go
// prefix: KEEL_MEMORY_
type Config struct {
    GRPCPort              int     `envconfig:"GRPC_PORT"               default:"50053"`
    HealthPort            int     `envconfig:"HEALTH_PORT"             default:"8080"`
    MetricsPort           int     `envconfig:"METRICS_PORT"            default:"9090"`
    DatabaseURL           string  `envconfig:"DATABASE_URL"            required:"true"`
    RedisURL              string  `envconfig:"REDIS_URL"               required:"true"`
    DefaultVectorDims     int     `envconfig:"DEFAULT_VECTOR_DIMS"     default:"1536"`
    MaxTopK               int     `envconfig:"MAX_TOP_K"               default:"100"`
    MinSimilarity         float32 `envconfig:"MIN_SIMILARITY"          default:"0.7"`
    MaxNamespaceEntries   int     `envconfig:"MAX_NAMESPACE_ENTRIES"   default:"100000"`
    ShutdownGraceSecs     int     `envconfig:"SHUTDOWN_GRACE_SECS"     default:"30"`
    LogLevel              string  `envconfig:"LOG_LEVEL"               default:"INFO"`
    OtelEndpoint          string  `envconfig:"OTEL_ENDPOINT"           default:"http://otel-collector:4317"`
    ServiceName           string  `envconfig:"SERVICE_NAME"            default:"memory-service"`
}
```

---

## BDD Scenarios

```gherkin
Feature: Agent Memory

  Scenario: Set and retrieve a key-value entry
    Given namespace "agent:test-01" exists
    When entry key="ctx" value="hello" is set
    Then GetEntry for key="ctx" returns value="hello"

  Scenario: Entry expires after TTL
    Given an entry with ttl=1s is set in namespace "agent:ttl-test"
    When 2 seconds pass
    Then GetEntry returns NOT_FOUND

  Scenario: Cannot write to non-existent namespace
    When SetEntry is called for namespace "agent:ghost"
    Then the gRPC status is NOT_FOUND

  Scenario: Vector dimension mismatch on upsert is rejected
    Given namespace "agent:vec-01" has vector_dimensions=1536
    When UpsertVector is called with an embedding of 768 dimensions
    Then the gRPC status is INVALID_ARGUMENT

  Scenario: Semantic search returns results ordered by similarity
    Given 3 vectors are upserted with known similarity scores [0.9, 0.7, 0.4]
    When SearchSimilar is called with top_k=2 and min_score=0.5
    Then 2 results are returned
    And the first result has the highest score

  Scenario: Namespace isolation — agent cannot read another agent's memory
    Given namespaces "agent:a1" and "agent:a2" each have an entry for key="secret"
    When GetEntry is called on "agent:a1" key="secret"
    Then only "agent:a1"'s value is returned
```
