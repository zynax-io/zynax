// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestValidateConfig_EmptyKey_ProductionMode_Fails(t *testing.T) {
	err := validateConfig(config{})
	if err == nil {
		t.Fatal("expected error for empty API key in production mode, got nil")
	}
}

func TestValidateConfig_EmptyKey_DevInsecure_OK(t *testing.T) {
	err := validateConfig(config{DevInsecure: true})
	if err != nil {
		t.Fatalf("expected no error in dev-insecure mode, got: %v", err)
	}
}

func TestValidateConfig_NonEmptyKey_OK(t *testing.T) {
	err := validateConfig(config{APIKey: "test-secret"})
	if err != nil {
		t.Fatalf("expected no error with API key set, got: %v", err)
	}
}
