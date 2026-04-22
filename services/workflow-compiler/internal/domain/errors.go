// Package domain contains the pure business logic for the workflow compiler.
// It has zero imports from the api or infrastructure layers.
package domain

import "fmt"

// CompilationErrorCode classifies structural and syntactic errors.
// Values mirror CompilationErrorCode in workflow_compiler.proto and are
// permanent — the api layer maps domain codes to proto codes.
type CompilationErrorCode int

const (
	ErrorCodeUnspecified           CompilationErrorCode = 0
	ErrorCodeYAMLParseError        CompilationErrorCode = 1
	ErrorCodeNoInitialState        CompilationErrorCode = 2
	ErrorCodeMultipleInitialStates CompilationErrorCode = 3
	ErrorCodeNoTerminalState       CompilationErrorCode = 4
	ErrorCodeOrphanState           CompilationErrorCode = 5
	ErrorCodeUnknownStateReference CompilationErrorCode = 6
	ErrorCodeDuplicateStateName    CompilationErrorCode = 7
	ErrorCodeMissingRequiredField  CompilationErrorCode = 8
	ErrorCodeInvalidFieldValue     CompilationErrorCode = 9
)

// ParseError describes a single structural or syntactic error found during
// manifest parsing or graph validation.
type ParseError struct {
	Code      CompilationErrorCode
	Message   string
	Line      int    // 1-based; zero when not attributable to a specific line
	StateName string // populated for state-scoped errors
}

func (e ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

// ParseErrors is a slice of ParseError that also implements the error interface.
type ParseErrors []ParseError

func (pe ParseErrors) Error() string {
	if len(pe) == 0 {
		return "no errors"
	}
	return pe[0].Error()
}
