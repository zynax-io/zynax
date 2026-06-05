// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// encodeCursor returns a stable, opaque page token for the (createdAt, taskID) keyset.
func encodeCursor(createdAt time.Time, taskID string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + taskID
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// decodeCursor parses a page token produced by encodeCursor.
func decodeCursor(token string) (time.Time, string, error) {
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("base64 decode: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("malformed cursor: missing separator")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse cursor time: %w", err)
	}
	return t, parts[1], nil
}

// marshalJSONBDetails encodes a string map to a JSONB-compatible byte slice.
func marshalJSONBDetails(m map[string]string) []byte {
	b, _ := json.Marshal(m)
	return b
}

// parseJSONBDetails decodes a JSONB byte slice back to a string map.
func parseJSONBDetails(b []byte) map[string]string {
	var m map[string]string
	_ = json.Unmarshal(b, &m)
	return m
}
