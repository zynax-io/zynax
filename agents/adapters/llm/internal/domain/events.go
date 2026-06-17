// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"encoding/json"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// streamSink wraps an EventSink with terminal-event helpers. It centralises the
// contract rules: a leading PROGRESS precedes every terminal event, and a stream
// error after a deadline maps to TIMEOUT rather than UPSTREAM_ERROR.
type streamSink struct{ EventSink }

// send forwards an event to the underlying sink, normalising the wrapped gRPC
// transport error for wrapcheck.
func (s streamSink) send(ev *zynaxv1.TaskEvent) error {
	if err := s.Send(ev); err != nil {
		return err //nolint:wrapcheck // gRPC stream error already carries transport context
	}
	return nil
}

// ensureProgress emits a single empty PROGRESS when the provider produced no
// tokens, so the "at least one PROGRESS before terminal" invariant always holds.
func (s streamSink) ensureProgress(taskID string, emitted bool) error {
	if emitted {
		return nil
	}
	return s.send(progressEvent(taskID, nil))
}

// terminalTimeout emits a leading PROGRESS (if none was sent) then the FAILED
// TIMEOUT terminal event.
func (s streamSink) terminalTimeout(taskID string, emitted bool) error {
	if err := s.ensureProgress(taskID, emitted); err != nil {
		return err
	}
	return s.send(failedEvent(taskID, codeTimeout, "request exceeded timeout"))
}

// terminalForError maps a provider error to the correct terminal event: a
// deadline-induced failure becomes TIMEOUT, any other becomes UPSTREAM_ERROR.
// A leading PROGRESS is emitted first when none was sent.
func (s streamSink) terminalForError(streamCtx context.Context, taskID string, err error, emitted bool) error {
	if streamCtx.Err() != nil {
		return s.terminalTimeout(taskID, emitted)
	}
	if e := s.ensureProgress(taskID, emitted); e != nil {
		return e
	}
	return s.send(failedEvent(taskID, codeUpstream, err.Error()))
}

// progressEvent builds a PROGRESS TaskEvent carrying one token chunk.
func progressEvent(taskID string, chunk []byte) *zynaxv1.TaskEvent {
	return &zynaxv1.TaskEvent{
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS,
		Payload:   chunk,
		Timestamp: timestamppb.Now(),
	}
}

// completedEvent builds the terminal COMPLETED TaskEvent with the full response.
func completedEvent(taskID, full string) *zynaxv1.TaskEvent {
	payload, _ := json.Marshal(map[string]string{"completion": full})
	return &zynaxv1.TaskEvent{
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
		Payload:   payload,
		Timestamp: timestamppb.Now(),
	}
}

// failedEvent builds the terminal FAILED TaskEvent with a sanitised, truncated
// CapabilityError.message — never a raw body, credential, or stack trace.
func failedEvent(taskID, code, msg string) *zynaxv1.TaskEvent {
	return &zynaxv1.TaskEvent{
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
		Timestamp: timestamppb.Now(),
		Error:     &zynaxv1.CapabilityError{Code: code, Message: sanitise(msg)},
	}
}

// sanitise truncates a message to maxErrMsgLen runes. Callers must already have
// stripped credentials and raw bodies before passing a message here.
func sanitise(msg string) string {
	r := []rune(msg)
	if len(r) > maxErrMsgLen {
		return string(r[:maxErrMsgLen])
	}
	return msg
}
