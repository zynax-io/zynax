// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// AgentRepository persists and retrieves agent records.
// FindByCapability MUST return only REGISTERED agents.
type AgentRepository interface {
	Save(ctx context.Context, agent Agent) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (Agent, error)
	FindAll(ctx context.Context) ([]Agent, error)
	FindByCapability(ctx context.Context, name string) ([]Agent, error)
}
