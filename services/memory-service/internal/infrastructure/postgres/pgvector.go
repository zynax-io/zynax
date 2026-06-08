// SPDX-License-Identifier: Apache-2.0

// Package postgres provides a Postgres-backed VectorStore using pgx/v5 + pgvector.
package postgres

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // register pgx5 migrate driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// VectorStore is a pgx/v5 + pgvector implementation of domain.VectorStore.
// Namespace isolation is enforced via WHERE namespace = $N on every query.
type VectorStore struct {
	pool *pgxpool.Pool
}

// New opens a connection pool, runs pending migrations, and returns a VectorStore.
// dsn must never be hardcoded — read from ZYNAX_DB_DSN (env).
func New(ctx context.Context, dsn string) (*VectorStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("memory-service pgvector: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("memory-service pgvector: ping: %w", err)
	}
	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("memory-service pgvector: migrate: %w", err)
	}
	return &VectorStore{pool: pool}, nil
}

// Close releases the connection pool.
func (s *VectorStore) Close() {
	s.pool.Close()
}

// StoreVector upserts an embedding with the given id in namespace ns.
// metadata is serialised as JSONB. On conflict (namespace, id) the embedding and
// metadata are updated and updated_at is refreshed.
func (s *VectorStore) StoreVector(
	ctx context.Context,
	ns, id string,
	vector []float32,
	metadata map[string]string,
) error {
	metaJSON, err := marshalMetadata(metadata)
	if err != nil {
		return fmt.Errorf("memory-service pgvector: marshal metadata: %w", err)
	}

	vec := pgvector.NewVector(vector)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memory_vectors (id, namespace, embedding, metadata)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (namespace, id)
		DO UPDATE SET
			embedding  = EXCLUDED.embedding,
			metadata   = EXCLUDED.metadata,
			updated_at = now()
	`, id, ns, vec, metaJSON)
	if err != nil {
		return fmt.Errorf("memory-service pgvector: store vector %q/%q: %w", ns, id, err)
	}
	return nil
}

// QueryVector returns the k nearest embeddings in ns to the given query vector,
// ordered by cosine distance (ascending — most similar first).
// filter is reserved for future metadata predicate use; pass "" to disable.
func (s *VectorStore) QueryVector(
	ctx context.Context,
	ns string,
	vector []float32,
	k int,
	filter string,
) ([]domain.VectorResult, error) {
	if k <= 0 {
		return nil, fmt.Errorf("memory-service pgvector: k must be > 0")
	}

	vec := pgvector.NewVector(vector)

	// Build optional metadata filter clause.
	// filter format: "key=value" — simple equality on metadata JSONB key.
	// Empty string means no filter.
	filterClause, filterArgs := buildFilterClause(filter, ns, vec, k)

	rows, err := s.pool.Query(ctx, filterClause, filterArgs...)
	if err != nil {
		return nil, fmt.Errorf("memory-service pgvector: query vector in %q: %w", ns, err)
	}
	defer rows.Close()

	var results []domain.VectorResult
	for rows.Next() {
		var (
			id       string
			score    float64
			metaJSON []byte
		)
		if err := rows.Scan(&id, &score, &metaJSON); err != nil {
			return nil, fmt.Errorf("memory-service pgvector: scan row: %w", err)
		}
		meta, err := unmarshalMetadata(metaJSON)
		if err != nil {
			return nil, fmt.Errorf("memory-service pgvector: unmarshal metadata: %w", err)
		}
		results = append(results, domain.VectorResult{
			ID:       id,
			Score:    float32(score),
			Metadata: meta,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory-service pgvector: rows error: %w", err)
	}
	return results, nil
}

// DeleteVector removes the embedding identified by id in ns.
// No-op if the id does not exist.
func (s *VectorStore) DeleteVector(ctx context.Context, ns, id string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM memory_vectors WHERE namespace = $1 AND id = $2`,
		ns, id,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("memory-service pgvector: delete vector %q/%q: %w", ns, id, err)
	}
	return nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// buildFilterClause constructs the ANN query SQL and its argument list.
// If filter is non-empty and has the form "key=value", a JSONB equality predicate
// is added. This is intentionally minimal for M6 — extend in M7 if needed.
func buildFilterClause(filter, ns string, vec pgvector.Vector, k int) (string, []interface{}) {
	baseSQL := `
		SELECT id,
		       (embedding <=> $1) AS score,
		       metadata
		FROM   memory_vectors
		WHERE  namespace = $2`
	args := []interface{}{vec, ns}
	argIdx := 3

	if filter != "" {
		parts := strings.SplitN(filter, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			val := parts[1]
			baseSQL += fmt.Sprintf(` AND metadata->>'%s' = $%d`, sanitizeIdentifier(key), argIdx)
			args = append(args, val)
			argIdx++
		}
	}
	_ = argIdx
	// Use a literal integer for LIMIT — some Postgres HNSW planner versions
	// require a constant here to choose the ANN scan correctly.
	baseSQL += fmt.Sprintf(`
		ORDER BY embedding <=> $1
		LIMIT %d`, k)
	return baseSQL, args
}

// sanitizeIdentifier strips characters that are unsafe in a JSONB key reference.
// Only alphanumeric + underscore allowed; everything else is removed.
func sanitizeIdentifier(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func marshalMetadata(m map[string]string) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func unmarshalMetadata(data []byte) (map[string]string, error) {
	if len(data) == 0 {
		return map[string]string{}, nil
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func runMigrations(dsn string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+stripSchema(dsn))
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// stripSchema removes a leading "postgres://" or "postgresql://" prefix so that
// golang-migrate receives the "pgx5://" scheme it expects.
func stripSchema(dsn string) string {
	for _, pfx := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(pfx) && dsn[:len(pfx)] == pfx {
			return dsn[len(pfx):]
		}
	}
	return dsn
}
