// Package api implements the WorkflowCompilerService gRPC server.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/ir"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/validators"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements WorkflowCompilerServiceServer.
// The compiler is stateless: each CompileWorkflow call compiles the manifest
// and returns the IR in the response. Callers must retain ir_payload if they
// need the IR after the RPC — GetCompiledWorkflow always returns NOT_FOUND.
//
// Server is not usable at zero value; always construct via New().
type Server struct {
	zynaxv1.UnimplementedWorkflowCompilerServiceServer
	generateID func() string
	policyGate *domain.PolicyGate // nil → policy enforcement disabled
}

// New creates a Server ready to serve gRPC requests.
func New() *Server {
	return &Server{generateID: generateWorkflowID}
}

// NewWithPolicy creates a Server with a PolicyGate that enforces routing
// policies and capability quotas at compile time. Pass nil to disable
// policy enforcement (equivalent to New()).
func NewWithPolicy(gate *domain.PolicyGate) *Server {
	return &Server{generateID: generateWorkflowID, policyGate: gate}
}

// CompileWorkflow parses, validates, and compiles a YAML manifest into a WorkflowIR.
// The compiled IR is returned in the response; it is not stored anywhere.
func (s *Server) CompileWorkflow(ctx context.Context, req *zynaxv1.CompileWorkflowRequest) (*zynaxv1.CompileWorkflowResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if len(req.ManifestYaml) == 0 {
		return nil, status.Error(codes.InvalidArgument, "manifest_yaml must not be empty")
	}

	start := time.Now()

	manifest, parseErrs := domain.ParseManifest(ctx, req.ManifestYaml)
	if len(parseErrs) > 0 {
		return &zynaxv1.CompileWorkflowResponse{
			Errors: toProtoErrors(parseErrs),
		}, nil
	}

	if manifest.Namespace == "" && req.Namespace != "" {
		manifest.Namespace = req.Namespace
	}

	// Policy gate: routing policy and capability quota enforcement.
	// Runs after parsing (so we know namespace + annotations) and before the
	// graph build (fail fast on policy violations).
	if s.policyGate != nil {
		annotations := manifest.Annotations
		if annotations == nil {
			annotations = map[string]string{}
		}
		// Build a minimal graph-like struct for the gate — only Namespace is
		// needed at this stage; the full graph is built next.
		stub := &domain.WorkflowGraph{Namespace: manifest.Namespace}
		if gateErr := s.policyGate.Check(ctx, stub, annotations); gateErr != nil {
			switch gateErr.Kind {
			case domain.PolicyViolationRouting:
				return nil, status.Errorf(codes.PermissionDenied,
					"routing policy violation: %s", gateErr.Message)
			case domain.PolicyViolationQuota:
				return nil, status.Errorf(codes.ResourceExhausted,
					"capability quota exceeded: %s", gateErr.Message)
			default:
				return nil, status.Errorf(codes.Internal,
					"policy gate error: %s", gateErr.Message)
			}
		}
	}

	g, buildErrs := domain.Build(ctx, manifest)
	if len(buildErrs) > 0 {
		return &zynaxv1.CompileWorkflowResponse{
			Errors: toProtoErrors(buildErrs),
		}, nil
	}

	if validationErrs := validators.Run(ctx, g, validators.All()...); len(validationErrs) > 0 {
		return &zynaxv1.CompileWorkflowResponse{
			Errors: toProtoErrors(validationErrs),
		}, nil
	}

	wfID := s.generateID()
	wfIR, err := ir.ToIR(ctx, g, wfID, manifest.APIVersion, time.Now().UTC())
	if err != nil {
		return nil, grpcErr(fmt.Errorf("IR generation: %w", err))
	}

	durationMs := time.Since(start).Milliseconds()
	if durationMs == 0 {
		durationMs = minDurationMs
	}

	return &zynaxv1.CompileWorkflowResponse{
		WorkflowIr:            wfIR,
		CompilationDurationMs: durationMs,
	}, nil
}

// ValidateManifest checks a manifest for structural correctness without persisting anything.
// All errors found are returned — not just the first.
func (s *Server) ValidateManifest(ctx context.Context, req *zynaxv1.ValidateManifestRequest) (*zynaxv1.ValidateManifestResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if len(req.ManifestYaml) == 0 {
		return nil, status.Error(codes.InvalidArgument, "manifest_yaml must not be empty")
	}

	manifest, parseErrs := domain.ParseManifest(ctx, req.ManifestYaml)
	if len(parseErrs) > 0 {
		return &zynaxv1.ValidateManifestResponse{
			Valid:  false,
			Errors: toProtoErrors(parseErrs),
		}, nil
	}

	var allErrs []domain.ParseError

	g, buildErrs := domain.Build(ctx, manifest)
	allErrs = append(allErrs, buildErrs...)

	if g != nil {
		allErrs = append(allErrs, validators.Run(ctx, g, validators.All()...)...)
	}

	return &zynaxv1.ValidateManifestResponse{
		Valid:  len(allErrs) == 0,
		Errors: toProtoErrors(allErrs),
	}, nil
}

// GetCompiledWorkflow always returns NOT_FOUND. The compiler is stateless — the
// compiled IR is returned in the CompileWorkflow response and is not stored.
// Callers must retain the ir_payload from CompileWorkflowResponse and pass it
// directly to EngineAdapterService.SubmitWorkflow.
func (s *Server) GetCompiledWorkflow(ctx context.Context, req *zynaxv1.GetCompiledWorkflowRequest) (*zynaxv1.GetCompiledWorkflowResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	return nil, status.Errorf(codes.NotFound,
		"workflow %q not found: compiler is stateless — retain ir_payload from CompileWorkflow response",
		req.WorkflowId,
	)
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
		var lineNumber int32
		if e.Line <= math.MaxInt32 {
			lineNumber = int32(e.Line) //nolint:gosec // bounds-checked by enclosing if
		} else {
			lineNumber = math.MaxInt32
		}
		out = append(out, &zynaxv1.CompilationError{
			Code:       pbCode,
			Message:    e.Message,
			LineNumber: lineNumber,
			StateName:  e.StateName,
		})
	}
	return out
}

// minDurationMs is the minimum value reported for compilation_duration_ms.
// Prevents a zero duration from being misread as "not measured" by consumers.
const minDurationMs = 1

func generateWorkflowID() string {
	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes) // crypto/rand.Read never errors on supported platforms
	return fmt.Sprintf("wf-%s", hex.EncodeToString(randBytes))
}

// grpcErr maps a domain error to the appropriate gRPC status code.
// Input-validation guards (codes.InvalidArgument, codes.NotFound for in-memory
// lookups) stay inline in each handler; this helper handles unexpected domain
// failures surfaced as codes.Internal and context propagation errors.
func grpcErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, err.Error())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}
