// Package api implements the WorkflowCompilerService gRPC server.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/ir"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/validators"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements WorkflowCompilerServiceServer using in-memory IR storage.
// The in-memory store is appropriate for M2; a persistent backend is deferred.
type Server struct {
	zynaxv1.UnimplementedWorkflowCompilerServiceServer
	mu    sync.RWMutex
	store map[string]*zynaxv1.WorkflowIR
	idGen func() string
}

// New creates a Server ready to serve gRPC requests.
func New() *Server {
	return &Server{
		store: make(map[string]*zynaxv1.WorkflowIR),
		idGen: newWorkflowID,
	}
}

// CompileWorkflow parses, validates, and compiles a YAML manifest into a WorkflowIR.
// The compiled IR is stored unless dry_run is true.
func (s *Server) CompileWorkflow(_ context.Context, req *zynaxv1.CompileWorkflowRequest) (*zynaxv1.CompileWorkflowResponse, error) {
	if len(req.ManifestYaml) == 0 {
		return nil, status.Error(codes.InvalidArgument, "manifest_yaml must not be empty")
	}

	start := time.Now()

	manifest, parseErrs := domain.ParseManifest(req.ManifestYaml)
	if len(parseErrs) > 0 {
		return nil, status.Error(codes.InvalidArgument, parseErrs[0].Message)
	}

	if manifest.Namespace == "" && req.Namespace != "" {
		manifest.Namespace = req.Namespace
	}

	g, buildErrs := domain.Build(manifest)
	if len(buildErrs) > 0 {
		return nil, status.Error(codes.InvalidArgument, buildErrs[0].Message)
	}

	if validationErrs := validators.Run(g, validators.All()...); len(validationErrs) > 0 {
		return nil, status.Error(codes.InvalidArgument, validationErrs[0].Message)
	}

	wfID := s.idGen()
	wfIR, err := ir.ToIR(g, wfID, manifest.APIVersion, time.Now().UTC())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "IR generation: %v", err)
	}

	if !req.DryRun {
		s.mu.Lock()
		s.store[wfID] = wfIR
		s.mu.Unlock()
	}

	durationMs := time.Since(start).Milliseconds()
	if durationMs == 0 {
		durationMs = 1
	}

	return &zynaxv1.CompileWorkflowResponse{
		WorkflowIr:            wfIR,
		CompilationDurationMs: durationMs,
	}, nil
}

// ValidateManifest checks a manifest for structural correctness without persisting anything.
// All errors found are returned — not just the first.
func (s *Server) ValidateManifest(_ context.Context, req *zynaxv1.ValidateManifestRequest) (*zynaxv1.ValidateManifestResponse, error) {
	if len(req.ManifestYaml) == 0 {
		return nil, status.Error(codes.InvalidArgument, "manifest_yaml must not be empty")
	}

	manifest, parseErrs := domain.ParseManifest(req.ManifestYaml)
	if len(parseErrs) > 0 {
		return &zynaxv1.ValidateManifestResponse{
			Valid:  false,
			Errors: toProtoErrors(parseErrs),
		}, nil
	}

	var allErrs []domain.ParseError

	g, buildErrs := domain.Build(manifest)
	allErrs = append(allErrs, buildErrs...)

	if g != nil {
		allErrs = append(allErrs, validators.Run(g, validators.All()...)...)
	}

	return &zynaxv1.ValidateManifestResponse{
		Valid:  len(allErrs) == 0,
		Errors: toProtoErrors(allErrs),
	}, nil
}

// GetCompiledWorkflow retrieves a previously compiled WorkflowIR by workflow_id.
func (s *Server) GetCompiledWorkflow(_ context.Context, req *zynaxv1.GetCompiledWorkflowRequest) (*zynaxv1.GetCompiledWorkflowResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}

	s.mu.RLock()
	wfIR, ok := s.store[req.WorkflowId]
	s.mu.RUnlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "workflow %q not found", req.WorkflowId)
	}

	return &zynaxv1.GetCompiledWorkflowResponse{
		WorkflowIr: wfIR,
		CompiledAt: wfIR.CompiledAt,
	}, nil
}

// errorCodeMap maps domain codes to proto codes.
// ErrorCodeCircularTransition has no proto equivalent yet; maps to INVALID_FIELD_VALUE.
var errorCodeMap = map[domain.CompilationErrorCode]zynaxv1.CompilationErrorCode{
	domain.ErrorCodeUnspecified:           zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_UNSPECIFIED,
	domain.ErrorCodeYAMLParseError:        zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_YAML_PARSE_ERROR,
	domain.ErrorCodeNoInitialState:        zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_NO_INITIAL_STATE,
	domain.ErrorCodeMultipleInitialStates: zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_MULTIPLE_INITIAL_STATES,
	domain.ErrorCodeNoTerminalState:       zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_NO_TERMINAL_STATE,
	domain.ErrorCodeOrphanState:           zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_ORPHAN_STATE,
	domain.ErrorCodeUnknownStateReference: zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_UNKNOWN_STATE_REFERENCE,
	domain.ErrorCodeDuplicateStateName:    zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_DUPLICATE_STATE_NAME,
	domain.ErrorCodeMissingRequiredField:  zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_MISSING_REQUIRED_FIELD,
	domain.ErrorCodeInvalidFieldValue:     zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_INVALID_FIELD_VALUE,
	domain.ErrorCodeCircularTransition:    zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_INVALID_FIELD_VALUE,
}

func toProtoErrors(errs []domain.ParseError) []*zynaxv1.CompilationError {
	out := make([]*zynaxv1.CompilationError, 0, len(errs))
	for _, e := range errs {
		pbCode, ok := errorCodeMap[e.Code]
		if !ok {
			pbCode = zynaxv1.CompilationErrorCode_COMPILATION_ERROR_CODE_UNSPECIFIED
		}
		out = append(out, &zynaxv1.CompilationError{
			Code:       pbCode,
			Message:    e.Message,
			LineNumber: int32(e.Line), //nolint:gosec // line numbers are small positive integers
			StateName:  e.StateName,
		})
	}
	return out
}

func newWorkflowID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never errors on supported platforms
	return fmt.Sprintf("wf-%s", hex.EncodeToString(b))
}
