// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"
)

const (
	testSvc = "api-gateway"
	testVer = "v0.6.0"
)

func resetMatrixFlags() {
	matrixPrefix = "ghcr.io/zynax-io/zynax"
	matrixService, matrixVersion, matrixExisting, matrixCandSHAs = "", "", "", ""
}

func TestRunReleaseMatrix_Match(t *testing.T) {
	resetMatrixFlags()
	defer resetMatrixFlags()
	matrixService, matrixVersion = testSvc, testVer
	matrixCandSHAs = "deadbeef00 cafef00d11"
	matrixExisting = "latest\nmain\nmain-cafef00d11"
	c, out := cmdWithIO(t, "")
	if err := runReleaseMatrix(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"src=ghcr.io/zynax-io/zynax/api-gateway:main-cafef00d11",
		"tgt=ghcr.io/zynax-io/zynax/api-gateway:v0.6.0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q, got:\n%s", want, got)
		}
	}
}

func TestRunReleaseMatrix_NoMatchExcludes(t *testing.T) {
	resetMatrixFlags()
	defer resetMatrixFlags()
	matrixService, matrixVersion = testSvc, testVer
	matrixCandSHAs = "deadbeef00"
	matrixExisting = "latest\nv0.5.0"
	c, out := cmdWithIO(t, "")
	if err := runReleaseMatrix(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("want empty stdout for excluded service, got:\n%s", out.String())
	}
}

func TestRunReleaseMatrix_MissingVersion(t *testing.T) {
	resetMatrixFlags()
	defer resetMatrixFlags()
	matrixService = testSvc
	matrixCandSHAs, matrixExisting = "abc", "main-abc"
	c, _ := cmdWithIO(t, "")
	if err := runReleaseMatrix(c, nil); err == nil {
		t.Error("want error for missing --version")
	}
}

func resetNotesFlags() {
	notesVersion, notesServices = "", ""
}

func TestRunReleaseNotes_OK(t *testing.T) {
	resetNotesFlags()
	defer resetNotesFlags()
	notesVersion = testVer
	notesServices = "api-gateway\nengine-adapter"
	c, out := cmdWithIO(t, "")
	if err := runReleaseNotes(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "## Zynax v0.6.0") {
		t.Errorf("want notes header, got:\n%s", got)
	}
	if !strings.Contains(got, "docker pull ghcr.io/zynax-io/zynax/engine-adapter:v0.6.0") {
		t.Errorf("want engine-adapter pull line, got:\n%s", got)
	}
}

func TestRunReleaseNotes_MissingVersion(t *testing.T) {
	resetNotesFlags()
	defer resetNotesFlags()
	c, _ := cmdWithIO(t, "")
	if err := runReleaseNotes(c, nil); err == nil {
		t.Error("want error for missing --version")
	}
}
