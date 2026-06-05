// SPDX-License-Identifier: Apache-2.0

//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure/postgres"
)

func setupContainer(t *testing.T) (repo *postgres.AgentRepository, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("agent_registry_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}

	r, err := postgres.New(ctx, dsn)
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("open repository: %v", err)
	}

	return r, func() {
		r.Close()
		_ = ctr.Terminate(ctx)
	}
}

func makeAgent(id string, caps ...string) domain.Agent {
	capabilities := make([]domain.Capability, 0, len(caps))
	for _, c := range caps {
		capabilities = append(capabilities, domain.Capability{Name: c, Description: "test"})
	}
	return domain.Agent{
		ID:           id,
		Name:         "agent-" + id,
		Description:  "test agent",
		Endpoint:     "localhost:500" + id,
		Capabilities: capabilities,
		Labels:       map[string]string{"env": "test"},
		Status:       domain.AgentStatusRegistered,
		RegisteredAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
}

func TestSaveAndFindByID(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	agent := makeAgent("60", "echo")
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != agent.ID {
		t.Errorf("ID: want %q got %q", agent.ID, got.ID)
	}
	if got.Status != domain.AgentStatusRegistered {
		t.Errorf("Status: want REGISTERED got %v", got.Status)
	}
	if len(got.Capabilities) != 1 || got.Capabilities[0].Name != "echo" {
		t.Errorf("Capabilities: want [{echo}] got %v", got.Capabilities)
	}
	if got.Labels["env"] != "test" {
		t.Errorf("Labels[env]: want test got %q", got.Labels["env"])
	}
}

func TestFindByID_NotFound(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()

	_, err := repo.FindByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected ErrAgentNotFound, got nil")
	}
}

func TestDelete(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	agent := makeAgent("61", "run")
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Delete(ctx, agent.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID(ctx, agent.ID)
	if err == nil {
		t.Fatal("expected ErrAgentNotFound after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()

	err := repo.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected ErrAgentNotFound, got nil")
	}
}

func TestFindAll(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	for _, id := range []string{"62", "63"} {
		if err := repo.Save(ctx, makeAgent(id, "echo")); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("want 2 agents, got %d", len(all))
	}
}

func TestFindByCapability_RegisteredOnly(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	registered := makeAgent("64", "code-review")
	registered.Status = domain.AgentStatusRegistered
	if err := repo.Save(ctx, registered); err != nil {
		t.Fatalf("Save registered: %v", err)
	}

	deregistered := makeAgent("65", "code-review")
	deregistered.Status = domain.AgentStatusDeregistered
	if err := repo.Save(ctx, deregistered); err != nil {
		t.Fatalf("Save deregistered: %v", err)
	}

	results, err := repo.FindByCapability(ctx, "code-review")
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 REGISTERED agent, got %d", len(results))
	}
	if results[0].ID != registered.ID {
		t.Errorf("want agent %q, got %q", registered.ID, results[0].ID)
	}
}

func TestFindByCapability_MultipleCapabilities(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	a := makeAgent("66", "echo", "run", "build")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	for _, cap := range []string{"echo", "run", "build"} {
		results, err := repo.FindByCapability(ctx, cap)
		if err != nil {
			t.Fatalf("FindByCapability(%s): %v", cap, err)
		}
		if len(results) != 1 {
			t.Errorf("capability %q: want 1 agent, got %d", cap, len(results))
		}
	}

	results, err := repo.FindByCapability(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindByCapability(nonexistent): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for nonexistent capability, got %d", len(results))
	}
}

func TestSave_Upsert(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	agent := makeAgent("67", "echo")
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	agent.Status = domain.AgentStatusDeregistered
	agent.Name = "updated-name"
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("upsert Save: %v", err)
	}

	got, err := repo.FindByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("FindByID after upsert: %v", err)
	}
	if got.Status != domain.AgentStatusDeregistered {
		t.Errorf("Status: want DEREGISTERED got %v", got.Status)
	}
	if got.Name != "updated-name" {
		t.Errorf("Name: want updated-name got %q", got.Name)
	}
}

func TestMigration_Idempotent(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()

	agent := makeAgent("68", "echo")
	if err := repo.Save(context.Background(), agent); err != nil {
		t.Fatalf("Save after migration: %v", err)
	}
}
