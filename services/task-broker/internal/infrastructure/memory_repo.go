// SPDX-License-Identifier: Apache-2.0

// Package infrastructure contains adapters that implement domain ports.
package infrastructure

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

const (
	defaultPageSize = 50
	maxPageSize     = 500
)

type memoryRepo struct {
	mu    sync.RWMutex
	tasks map[string]*domain.Task
}

// NewMemoryRepo creates an in-memory TaskRepository.
func NewMemoryRepo() domain.TaskRepository {
	return &memoryRepo{tasks: make(map[string]*domain.Task)}
}

func (r *memoryRepo) Save(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.TaskID] = copyTask(task)
	return nil
}

func (r *memoryRepo) GetByID(_ context.Context, taskID string) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskNotFound, taskID)
	}
	return copyTask(t), nil
}

func (r *memoryRepo) Update(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[task.TaskID]; !ok {
		return fmt.Errorf("%w: %q", domain.ErrTaskNotFound, task.TaskID)
	}
	r.tasks[task.TaskID] = copyTask(task)
	return nil
}

func (r *memoryRepo) List(_ context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	var offset int
	if filter.PageToken != "" {
		var err error
		offset, err = decodePageToken(filter.PageToken)
		if err != nil {
			return domain.ListResult{}, fmt.Errorf("task-broker: invalid page_token: %w", err)
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []*domain.Task
	for _, t := range r.tasks {
		if filter.WorkflowID != "" && t.WorkflowID != filter.WorkflowID {
			continue
		}
		if filter.Status != domain.TaskStatusUnspecified && t.Status != filter.Status {
			continue
		}
		if filter.AgentID != "" && t.DispatchedTo != filter.AgentID {
			continue
		}
		matched = append(matched, copyTask(t))
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].TaskID < matched[j].TaskID
		}
		return matched[i].CreatedAt.Before(matched[j].CreatedAt)
	})

	total := len(matched)
	if offset >= total {
		return domain.ListResult{}, nil
	}

	end := offset + int(pageSize)
	hasMore := end < total
	if end > total {
		end = total
	}

	var nextPageToken string
	if hasMore {
		nextPageToken = encodePageToken(end)
	}

	return domain.ListResult{
		Tasks:         matched[offset:end],
		NextPageToken: nextPageToken,
	}, nil
}

func encodePageToken(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodePageToken(token string) (int, error) {
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("base64 decode: %w", err)
	}
	n, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, fmt.Errorf("parse offset: %w", err)
	}
	return n, nil
}

func copyTask(t *domain.Task) *domain.Task {
	c := *t
	if t.Error != nil {
		e := *t.Error
		if t.Error.Details != nil {
			e.Details = make(map[string]string, len(t.Error.Details))
			for k, v := range t.Error.Details {
				e.Details[k] = v
			}
		}
		c.Error = &e
	}
	if t.InputPayload != nil {
		c.InputPayload = make([]byte, len(t.InputPayload))
		copy(c.InputPayload, t.InputPayload)
	}
	if t.ResultPayload != nil {
		c.ResultPayload = make([]byte, len(t.ResultPayload))
		copy(c.ResultPayload, t.ResultPayload)
	}
	return &c
}
