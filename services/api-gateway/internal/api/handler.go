// SPDX-License-Identifier: Apache-2.0

// Package api implements the api-gateway HTTP handlers.
// It translates HTTP requests into domain calls and maps domain errors to
// HTTP status codes. No business logic lives here.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MB

// Handler handles HTTP requests for POST /api/v1/apply and GET /api/v1/workflows/{id}.
type Handler struct {
	svc *domain.ApplyService
}

// NewHandler creates a Handler backed by the given ApplyService.
func NewHandler(svc *domain.ApplyService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers all HTTP routes on mux. Requires Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/apply", h.handleApply)
	mux.HandleFunc("GET /api/v1/workflows/{id}/logs", h.handleWorkflowLogs)
	mux.HandleFunc("GET /api/v1/workflows/{id}", h.handleGetWorkflow)
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", h.handleDeleteWorkflow)
}

func (h *Handler) handleApply(w http.ResponseWriter, r *http.Request) {
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	kind, err := domain.DetectKind(body)
	if err != nil {
		code := "UNSUPPORTED_KIND"
		if !errors.Is(err, domain.ErrUnknownKind) {
			code = "INVALID_YAML"
		}
		writeError(w, http.StatusBadRequest, err.Error(), code)
		return
	}
	switch kind {
	case domain.KindWorkflow:
		h.applyWorkflow(w, r, body)
	case domain.KindAgentDef:
		h.applyAgentDef(w, r, body)
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("kind %q: not yet supported", kind), "UNSUPPORTED_KIND")
	}
}

func (h *Handler) applyWorkflow(w http.ResponseWriter, r *http.Request, body []byte) {
	req := domain.ApplyRequest{
		ManifestYAML: body,
		Namespace:    r.URL.Query().Get("namespace"),
		DryRun:       r.URL.Query().Get("dry_run") == "true",
		EngineHint:   r.URL.Query().Get("engine"),
	}
	result, err := h.svc.ApplyWorkflow(r.Context(), req)
	switch {
	case errors.Is(err, domain.ErrCompilationFailed):
		writeCompileErrors(w, result)
	case errors.Is(err, domain.ErrEngineUnavailable):
		writeError(w, http.StatusServiceUnavailable, "engine unavailable", "ENGINE_UNAVAILABLE")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL")
	case req.DryRun:
		writeJSON(w, http.StatusOK, dryRunResp{DryRun: true, Warnings: result.Warnings})
	default:
		writeJSON(w, http.StatusAccepted, applyResp{RunID: result.RunID, Warnings: result.Warnings})
	}
}

func (h *Handler) applyAgentDef(w http.ResponseWriter, r *http.Request, body []byte) {
	req := domain.ApplyRequest{
		ManifestYAML: body,
		Namespace:    r.URL.Query().Get("namespace"),
	}
	result, err := h.svc.ApplyAgentDef(r.Context(), req)
	switch {
	case errors.Is(err, domain.ErrAgentAlreadyExists):
		writeError(w, http.StatusConflict, "agent already registered", "ALREADY_EXISTS")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL")
	default:
		writeJSON(w, http.StatusCreated, agentDefResp{AgentID: result.AgentID})
	}
}

func (h *Handler) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "run_id is required", "INVALID_ARGUMENT")
		return
	}
	run, err := h.svc.GetWorkflowStatus(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "workflow run not found", "NOT_FOUND")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL")
	default:
		writeJSON(w, http.StatusOK, workflowStatusResp{
			RunID:        run.RunID,
			WorkflowID:   run.WorkflowID,
			Status:       run.Status,
			CurrentState: run.CurrentState,
		})
	}
}

func (h *Handler) handleWorkflowLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "run_id is required", "INVALID_ARGUMENT")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "INTERNAL")
		return
	}
	started := false
	err := h.svc.WatchWorkflowLogs(r.Context(), id, func(ev domain.WatchEvent) error {
		if !started {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
			started = true
		}
		data, merr := json.Marshal(sseWatchEvent(ev))
		if merr != nil {
			return merr
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	})
	if started {
		return
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "workflow run not found", "NOT_FOUND")
	case err != nil && !errors.Is(err, context.Canceled):
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL")
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "run_id is required", "INVALID_ARGUMENT")
		return
	}
	err := h.svc.CancelWorkflow(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "workflow run not found", "NOT_FOUND")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// ── response types ────────────────────────────────────────────────────────

type errResp struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type applyResp struct {
	RunID    string   `json:"run_id"`
	Warnings []string `json:"warnings,omitempty"`
}

type dryRunResp struct {
	DryRun   bool     `json:"dry_run"`
	Warnings []string `json:"warnings,omitempty"`
}

type compileErrItem struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Line    int32  `json:"line,omitempty"`
}

type compileErrsResp struct {
	Errors []compileErrItem `json:"errors"`
}

type agentDefResp struct {
	AgentID string `json:"agent_id"`
}

type workflowStatusResp struct {
	RunID        string `json:"run_id"`
	WorkflowID   string `json:"workflow_id"`
	Status       string `json:"status"`
	CurrentState string `json:"current_state"`
}

type watchEventResp struct {
	RunID     string `json:"run_id"`
	EventType string `json:"event_type"`
	FromState string `json:"from_state,omitempty"`
	ToState   string `json:"to_state,omitempty"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
	Payload   string `json:"payload,omitempty"`
}

func sseWatchEvent(ev domain.WatchEvent) watchEventResp {
	return watchEventResp{
		RunID:     ev.RunID,
		EventType: ev.EventType,
		FromState: ev.FromState,
		ToState:   ev.ToState,
		Status:    ev.Status,
		Timestamp: ev.Timestamp,
		Payload:   ev.Payload,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large", "BODY_TOO_LARGE")
		return nil, false
	}
	return body, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, errResp{Error: message, Code: code})
}

func writeCompileErrors(w http.ResponseWriter, result domain.ApplyResult) {
	items := make([]compileErrItem, len(result.Errors))
	for i, e := range result.Errors {
		items[i] = compileErrItem{Code: e.Code, Message: e.Message, Line: e.Line}
	}
	writeJSON(w, http.StatusUnprocessableEntity, compileErrsResp{Errors: items})
}
