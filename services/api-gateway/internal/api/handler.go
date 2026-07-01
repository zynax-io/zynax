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
	"time"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MB

// Handler handles HTTP requests for POST /api/v1/apply and GET /api/v1/workflows/{id}.
type Handler struct {
	svc    *domain.ApplyService
	apiKey string
}

// NewHandler creates a Handler backed by the given ApplyService.
// apiKey is the value of ZYNAX_API_KEY; an empty string disables bearer auth.
func NewHandler(svc *domain.ApplyService, apiKey string) *Handler {
	return &Handler{svc: svc, apiKey: apiKey}
}

// RegisterRoutes registers all HTTP routes on mux. Requires Go 1.22+ ServeMux.
// Mutating endpoints (POST, DELETE) are protected by bearer-token auth when
// ZYNAX_API_KEY is set; read-only endpoints (GET) are always open.
// The mutating POST endpoints /api/v1/apply and /api/v1/workflows/{id}/events
// are additionally protected by a per-IP token-bucket rate limiter (see
// ratelimit.go); parameters are configured via RATE_LIMIT_RPS and
// RATE_LIMIT_BURST environment variables.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	rl := newIPRateLimiter()
	applyHandler := rl.Middleware(http.HandlerFunc(requireBearer(h.apiKey, h.handleApply)))
	mux.Handle("POST /api/v1/apply", applyHandler)
	mux.HandleFunc("GET /api/v1/workflows/{id}/logs", h.handleWorkflowLogs)
	mux.HandleFunc("GET /api/v1/workflows/{id}/outputs", h.handleWorkflowOutputs)
	eventsHandler := rl.Middleware(http.HandlerFunc(requireBearer(h.apiKey, h.handlePublishEvent)))
	mux.Handle("POST /api/v1/workflows/{id}/events", eventsHandler)
	mux.HandleFunc("GET /api/v1/workflows/{id}", h.handleGetWorkflow)
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", requireBearer(h.apiKey, h.handleDeleteWorkflow))
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
	// Attach the namespace to the request context so the downstream gRPC
	// interceptors carry it as correlation metadata on every hop (canvas C.2).
	// The X-Namespace header set by RequestIDMiddleware takes precedence; the
	// ?namespace= query param only applies when no header value is present, and
	// an empty param never clears an existing namespace.
	if ns := r.URL.Query().Get("namespace"); ns != "" && domain.NamespaceFromContext(r.Context()) == "" {
		r = r.WithContext(domain.WithNamespace(r.Context(), ns))
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
		writeJSON(w, http.StatusAccepted, applyResp{RunID: result.RunID, Status: result.Status, Warnings: result.Warnings})
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

// statusCompleted is the terminal status string for a successfully completed run
// (WorkflowStatus enum rendered via .String()).
const statusCompleted = "WORKFLOW_STATUS_COMPLETED"

// handleWorkflowOutputs serves the resolved workflow-level outputs of a run
// (ADR-042, M7.U). It returns the outputs JSON object for a COMPLETED run, an
// empty object {} for a run that declared none, and 404 for an unknown id.
// GET stays open (read-only). json.Encoder escapes control characters, so
// attacker-influenced output values render safely.
func (h *Handler) handleWorkflowOutputs(w http.ResponseWriter, r *http.Request) {
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
		outputs := run.Outputs
		if outputs == nil {
			outputs = map[string]string{}
		}
		writeJSON(w, http.StatusOK, outputs)
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
	// Disable the server-level WriteTimeout for this connection so long-running
	// workflows are not forcibly cut off at 30 s. Non-streaming endpoints keep
	// the server-wide deadline because they never call this code path.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	ctx := r.Context()
	started := false
	err := h.svc.WatchWorkflowLogs(ctx, id, func(ev domain.WatchEvent) error {
		if !started {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
			started = true
		}
		// On the terminal completed event, enrich the payload with the run's
		// declared outputs so streaming consumers get the result without a
		// separate /outputs read-back (ADR-042, M7.U O.8). The engine's history
		// stream does not carry the workflow result, so it is fetched here using
		// the same request-scoped context.
		if ev.Status == statusCompleted && ev.Payload == "" {
			if run, gerr := h.svc.GetWorkflowStatus(ctx, id); gerr == nil && len(run.Outputs) > 0 { //nolint:contextcheck // same request ctx captured above; the WatchWorkflowLogs callback carries no ctx param
				if b, merr := json.Marshal(map[string]map[string]string{"outputs": run.Outputs}); merr == nil {
					ev.Payload = string(b)
				}
			}
		}
		data, merr := json.Marshal(sseWatchEvent(ev))
		if merr != nil {
			return fmt.Errorf("api-gateway: marshal event: %w", merr)
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
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

// handlePublishEvent injects a business/lifecycle event into a workflow run so
// an event-driven workflow can advance. Body: {"event_type": "...", "data": {...}}.
// The optional data object is forwarded verbatim as the CloudEvent payload.
func (h *Handler) handlePublishEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "run_id is required", "INVALID_ARGUMENT")
		return
	}
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	var req publishEventReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}
	eventID, err := h.svc.PublishEvent(r.Context(), domain.EventPublish{
		RunID: id,
		Type:  req.EventType,
		Data:  req.Data,
	})
	switch {
	case errors.Is(err, domain.ErrInvalidEvent):
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_ARGUMENT")
	case errors.Is(err, domain.ErrEngineUnavailable):
		writeError(w, http.StatusServiceUnavailable, "event bus unavailable", "EVENT_BUS_UNAVAILABLE")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error", "INTERNAL")
	default:
		writeJSON(w, http.StatusAccepted, publishEventResp{EventID: eventID})
	}
}

// ── response types ────────────────────────────────────────────────────────

type errResp struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type applyResp struct {
	RunID    string   `json:"run_id"`
	Status   string   `json:"status,omitempty"`
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

type publishEventReq struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type publishEventResp struct {
	EventID string `json:"event_id,omitempty"`
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
