// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import "strings"

// streamSubjectDepth is the number of leading subject segments that identify
// the entity owning a JetStream stream, per the platform topic taxonomy
// "zynax.<version>.<service>.<entity>.<event_type>" (root AGENTS.md): the
// entity prefix is always the first four segments, while the event_type verb
// may itself contain dots (e.g. "state.entered").
//
// Deriving every stream at the same fixed depth makes subject filters either
// identical (same stream) or pairwise disjoint by construction. The previous
// "drop the last segment" rule derived overlapping filters for event types of
// different depth within the same prefix tree (e.g. "….workflow.state.entered"
// vs "….workflow.completed"), and NATS rejected the second stream with
// "subjects overlap with an existing stream" (err 10065), silently making
// whole event families undeliverable — see #1149.
const streamSubjectDepth = 4

// streamPrefix returns the leading subject segments (at most
// streamSubjectDepth) that identify the stream owning eventType. Event types
// shorter than the taxonomy depth are used verbatim.
func streamPrefix(eventType string) string {
	parts := strings.Split(eventType, ".")
	if len(parts) > streamSubjectDepth {
		return strings.Join(parts[:streamSubjectDepth], ".")
	}
	return eventType
}

// StreamName derives a JetStream stream name from a dot-separated event type.
// Examples:
//
//	"zynax.v1.engine-adapter.workflow.completed"     → "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
//	"zynax.v1.engine-adapter.workflow.state.entered" → "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
//	"zynax.v1.agent-registry.agent.registered"       → "ZYNAX_V1_AGENT_REGISTRY_AGENT"
//
// Dashes are replaced with underscores; dots become underscores; all uppercase.
// All events under the same entity prefix (first streamSubjectDepth segments)
// share a single stream regardless of how many segments the verb has.
func StreamName(eventType string) string {
	name := strings.ReplaceAll(streamPrefix(eventType), ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToUpper(name)
}

// SubjectFilter returns the widest NATS subject filter for the stream that
// owns eventType: "<entity-prefix>.>" for taxonomy-depth event types, or the
// literal event type when it has fewer segments than the taxonomy depth
// (literal subjects can never overlap a fixed-depth wildcard filter).
func SubjectFilter(eventType string) string {
	prefix := streamPrefix(eventType)
	if prefix != eventType {
		return prefix + ".>"
	}
	return eventType
}

// streamSubjects returns the full subject set for the stream owning eventType.
// Taxonomy-depth streams carry both the literal entity prefix and its ".>"
// wildcard so a (degenerate) event type that IS the entity prefix still lands
// on the same stream instead of requiring an overlapping second stream.
func streamSubjects(eventType string) []string {
	prefix := streamPrefix(eventType)
	if prefix != eventType {
		return []string{prefix, prefix + ".>"}
	}
	return []string{eventType}
}

// DurableConsumerName converts a subscriber_id into a valid JetStream durable
// consumer name. JetStream consumer names may not contain spaces, dots, or
// special characters; we replace every non-alphanumeric-or-dash character with
// an underscore and truncate at 200 bytes to stay under the NATS limit.
func DurableConsumerName(subscriberID string) string {
	var b strings.Builder
	for _, r := range subscriberID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	name := b.String()
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

// MatchesGlob reports whether eventType matches a glob pattern where
// "*" matches exactly one dot-separated segment and "**" matches zero or more
// dot-separated segments.
func MatchesGlob(pattern, eventType string) bool {
	return matchGlobSegments(strings.Split(pattern, "."), strings.Split(eventType, "."))
}

func matchGlobSegments(pat, seg []string) bool {
	for len(pat) > 0 {
		p := pat[0]
		if p == "**" {
			// "**" at end matches everything remaining (zero or more segments).
			if len(pat) == 1 {
				return true
			}
			// Try matching the rest of the pattern against every suffix of seg (including empty).
			rest := pat[1:]
			for j := 0; j <= len(seg); j++ {
				if matchGlobSegments(rest, seg[j:]) {
					return true
				}
			}
			return false
		}
		if len(seg) == 0 {
			return false
		}
		if p != "*" && p != seg[0] {
			return false
		}
		pat = pat[1:]
		seg = seg[1:]
	}
	return len(seg) == 0
}

// StreamSubjectFromPattern extracts a concrete subject from a glob pattern so
// we can create/reuse the correct JetStream stream.
// Examples:
//
//	"zynax.v1.engine-adapter.workflow.*" → "zynax.v1.engine-adapter.workflow.x"
//	"zynax.v1.**"                         → "zynax.v1.x"
//	"zynax.v1.workflow.completed"         → "zynax.v1.workflow.completed"
func StreamSubjectFromPattern(pattern string) string {
	parts := strings.Split(pattern, ".")
	concrete := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "*" || p == "**" {
			concrete = append(concrete, "x")
			break
		}
		concrete = append(concrete, p)
	}
	return strings.Join(concrete, ".")
}
