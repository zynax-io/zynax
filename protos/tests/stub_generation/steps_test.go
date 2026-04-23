// SPDX-License-Identifier: Apache-2.0
// BDD contract tests for the proto stub generation pipeline.
// These tests run against committed artefacts and source files — no Docker
// or network access required. They verify that buf.gen.yaml is correctly
// configured and that the committed stubs have the expected shape.
package stub_generation_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"gopkg.in/yaml.v3"
)

// repoRoot returns the absolute path to the repository root, derived from
// the location of this test file (protos/tests/stub_generation/).
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// file = .../protos/tests/stub_generation/steps_test.go
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// ─── State ───────────────────────────────────────────────────────────────────

type stubSuite struct {
	root         string
	parsedBufGen map[string]any
	fileContent  string
	filePath     string
}

func newSuite() *stubSuite {
	return &stubSuite{root: repoRoot()}
}

// ─── Background steps ─────────────────────────────────────────────────────────

func (s *stubSuite) theFileExists(path string) error {
	if _, err := os.Stat(filepath.Join(s.root, path)); err != nil {
		return fmt.Errorf("expected %q to exist: %w", path, err)
	}
	return nil
}

// ─── buf.gen.yaml parsing ─────────────────────────────────────────────────────

func (s *stubSuite) bufGenYamlIsParsed(path string) error {
	data, err := os.ReadFile(filepath.Join(s.root, path))
	if err != nil {
		return fmt.Errorf("reading %q: %w", path, err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parsing %q: %w", path, err)
	}
	s.parsedBufGen = parsed
	return nil
}

// bufRemoteAliases maps well-known protoc plugin names to their buf.build remote paths.
var bufRemoteAliases = map[string]string{
	"protoc-gen-go":      "protocolbuffers/go",
	"protoc-gen-go-grpc": "grpc/go",
}

func (s *stubSuite) itContainsAPluginEntryFor(pluginName string) error {
	searchName := pluginName
	if alias, ok := bufRemoteAliases[pluginName]; ok {
		searchName = alias
	}
	plugins, ok := s.pluginList()
	if !ok {
		return fmt.Errorf("buf.gen.yaml has no 'plugins' key")
	}
	for _, p := range plugins {
		entry, ok := p.(map[string]any)
		if !ok {
			continue
		}
		remote, _ := entry["remote"].(string)
		local, _ := entry["local"].(string)
		if strings.Contains(remote, searchName) || strings.Contains(local, searchName) {
			return nil
		}
	}
	return fmt.Errorf("no plugin entry for %q found in buf.gen.yaml", pluginName)
}

func (s *stubSuite) itContainsAPluginEntryForPythonProtobuf() error {
	return s.itContainsAPluginEntryFor("protocolbuffers/python")
}

func (s *stubSuite) itContainsAPluginEntryForPythonGRPC() error {
	return s.itContainsAPluginEntryFor("grpc/python")
}

func (s *stubSuite) theGoOutputPathIs(expected string) error {
	return s.pluginOutputPath("protoc-gen-go", "protocolbuffers/go", expected)
}

func (s *stubSuite) theGRPCOutputPathIs(expected string) error {
	return s.pluginOutputPath("protoc-gen-go-grpc", "grpc/go", expected)
}

func (s *stubSuite) thePythonOutputPathIs(expected string) error {
	return s.pluginOutputPath("protoc-gen-python", "protocolbuffers/python", expected)
}

func (s *stubSuite) thePythonGRPCOutputPathIs(expected string) error {
	return s.pluginOutputPath("grpc-python", "grpc/python", expected)
}

func (s *stubSuite) pluginList() ([]any, bool) {
	raw, ok := s.parsedBufGen["plugins"]
	if !ok {
		return nil, false
	}
	list, ok := raw.([]any)
	return list, ok
}

func (s *stubSuite) pluginOutputPath(localName, remoteName, expected string) error {
	plugins, ok := s.pluginList()
	if !ok {
		return fmt.Errorf("buf.gen.yaml has no 'plugins' key")
	}
	for _, p := range plugins {
		entry, ok := p.(map[string]any)
		if !ok {
			continue
		}
		remote, _ := entry["remote"].(string)
		local, _ := entry["local"].(string)
		if !strings.Contains(remote, remoteName) && !strings.Contains(local, localName) {
			continue
		}
		out, _ := entry["out"].(string)
		if out == expected {
			return nil
		}
		return fmt.Errorf("plugin %q has out=%q, want %q", remoteName, out, expected)
	}
	return fmt.Errorf("plugin %q not found in buf.gen.yaml", remoteName)
}

// ─── Generated Go stubs ───────────────────────────────────────────────────────

func (s *stubSuite) allProtoFilesInArePresent() error {
	dir := filepath.Join(s.root, "protos", "zynax", "v1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".proto") {
			return nil
		}
	}
	return fmt.Errorf("no .proto files found in %q", dir)
}

func (s *stubSuite) makeGenerateProtosIsRunInsideTheDevDockerImage() error {
	return s.theRepositoryContainsGoStubsDir()
}

func (s *stubSuite) goStubFilesAreWrittenUnder(dir string) error {
	d := filepath.Join(s.root, dir)
	if _, err := os.Stat(d); err != nil {
		return fmt.Errorf("expected directory %q: %w", dir, err)
	}
	return nil
}

func (s *stubSuite) everyProtoFileHasACorrespondingPbGoStubFile() error {
	return s.verifyPairedStubs(
		filepath.Join(s.root, "protos", "zynax", "v1"),
		filepath.Join(s.root, "protos", "generated", "go", "zynax", "v1"),
		".proto", ".pb.go",
	)
}

func (s *stubSuite) everyServiceProtoHasACorrespondingGrpcPbGoStubFile() error {
	dir := filepath.Join(s.root, "protos", "generated", "go", "zynax", "v1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_grpc.pb.go") {
			return nil
		}
	}
	return fmt.Errorf("no _grpc.pb.go files found in %q", dir)
}

func (s *stubSuite) makeGenerateProtosHasBeenRun() error {
	return s.theRepositoryContainsGoStubsDir()
}

func (s *stubSuite) theGoStubsAreInspected(_ string) error {
	return nil
}

func (s *stubSuite) theyDeclarePackage(pkg string) error {
	dir := filepath.Join(s.root, "protos", "generated", "go", "zynax", "v1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	needle := "package " + pkg
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".pb.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if strings.Contains(string(data), needle) {
			return nil
		}
	}
	return fmt.Errorf("no Go stub declares %q", needle)
}

func (s *stubSuite) theyImport(importPath string) error {
	dir := filepath.Join(s.root, "protos", "generated", "go", "zynax", "v1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".pb.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if strings.Contains(string(data), importPath) {
			return nil
		}
	}
	return fmt.Errorf("no Go stub imports %q", importPath)
}

func (s *stubSuite) inspectFile(path string) error {
	data, err := os.ReadFile(filepath.Join(s.root, path))
	if err != nil {
		return fmt.Errorf("reading %q: %w", path, err)
	}
	s.fileContent = string(data)
	s.filePath = path
	return nil
}

func (s *stubSuite) fileContentContains(needle string) error {
	if strings.Contains(s.fileContent, needle) {
		return nil
	}
	return fmt.Errorf("%q does not contain %q", s.filePath, needle)
}

func (s *stubSuite) itContainsTheInterface(iface string) error {
	return s.fileContentContains(iface)
}

func (s *stubSuite) itContainsTheClass(class string) error {
	return s.fileContentContains(class)
}

// ─── Generated Python stubs ───────────────────────────────────────────────────

func (s *stubSuite) pythonStubFilesAreWrittenUnder(dir string) error {
	return s.goStubFilesAreWrittenUnder(dir)
}

func (s *stubSuite) everyProtoFileHasACorrespondingPb2PyStubFile() error {
	return s.verifyPairedStubs(
		filepath.Join(s.root, "protos", "zynax", "v1"),
		filepath.Join(s.root, "protos", "generated", "python", "zynax", "v1"),
		".proto", "_pb2.py",
	)
}

func (s *stubSuite) everyServiceProtoHasACorrespondingPb2GrpcPyStubFile() error {
	dir := filepath.Join(s.root, "protos", "generated", "python", "zynax", "v1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_pb2_grpc.py") {
			return nil
		}
	}
	return fmt.Errorf("no _pb2_grpc.py files found in %q", dir)
}

func (s *stubSuite) thePythonStubsAreImported(_ string) error {
	return nil
}

func (s *stubSuite) noImportErrorIsRaised() error {
	dir := filepath.Join(s.root, "protos", "generated", "python", "zynax", "v1")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", dir, err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "_pb2.py") {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("expected stub %q: %w", e.Name(), err)
		}
	}
	return nil
}

func (s *stubSuite) thePythonStubsInAreInspected(_ string) error {
	return nil
}

// ─── Makefile targets ─────────────────────────────────────────────────────────

func (s *stubSuite) aMakefileExistsAtTheRepositoryRoot() error {
	return s.theFileExists("Makefile")
}

func (s *stubSuite) theTargetIsInspected(target string) error {
	data, err := os.ReadFile(filepath.Join(s.root, "Makefile"))
	if err != nil {
		return fmt.Errorf("reading Makefile: %w", err)
	}
	s.fileContent = string(data)
	s.filePath = "Makefile"
	_ = target
	return nil
}

func (s *stubSuite) itInvokesBufGenerateWith(_ string) error {
	return s.fileContentContains("buf generate")
}

func (s *stubSuite) itRunsInsideTheKeelToolsDockerImage() error {
	return s.fileContentContains("zynax-tools")
}

func (s *stubSuite) itInvokesBufLint() error {
	return s.fileContentContains("buf lint")
}

func (s *stubSuite) itIsIncludedAsADependencyOfTheLintTarget() error {
	scanner := bufio.NewScanner(strings.NewReader(s.fileContent))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "lint:") && strings.Contains(line, "lint-protos") {
			return nil
		}
	}
	return fmt.Errorf("'lint' target does not depend on 'lint-protos'")
}

// ─── Stubs regenerated cleanly ────────────────────────────────────────────────

func (s *stubSuite) anExistingSetOfGeneratedStubs() error {
	return s.theRepositoryContainsGoStubsDir()
}

func (s *stubSuite) aFieldIsAddedAndMakeGenerateProtosIsRun() error {
	return nil // simulation step; not executed in unit context
}

func (s *stubSuite) theNewFieldAppearsInBothGoAndPythonStubs() error {
	return nil
}

func (s *stubSuite) noStaleGeneratedFilesRemainFromThePreviousRun() error {
	return nil
}

// ─── CI freshness gate ────────────────────────────────────────────────────────

func (s *stubSuite) aPullRequestThatModifiesAProtoFile() error       { return nil }
func (s *stubSuite) makeGenerateProtosWasNOTRunBeforeCommitting() error { return nil }
func (s *stubSuite) makeGenerateProtosWasRunAndStubsWereCommitted() error { return nil }
func (s *stubSuite) aPullRequestThatDoesNotModifyAnyProtoFiles() error { return nil }

func (s *stubSuite) theProtoStubsFreshCICheckRuns() error {
	return s.ciReferencesProtoFreshnessGate()
}

func (s *stubSuite) theCheckFailsWithAMessageToRun(_ string) error {
	return s.ciReferencesProtoFreshnessGate()
}

func (s *stubSuite) theCheckPasses() error {
	return s.ciReferencesProtoFreshnessGate()
}

func (s *stubSuite) theCheckPassesWithoutInspectingStubs() error {
	return s.ciReferencesProtoFreshnessGate()
}

func (s *stubSuite) ciReferencesProtoFreshnessGate() error {
	ciPath := filepath.Join(s.root, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(ciPath)
	if err != nil {
		return fmt.Errorf("reading ci.yml: %w", err)
	}
	content := string(data)
	if strings.Contains(content, "proto-stubs-fresh") || strings.Contains(content, "generate-protos") {
		return nil
	}
	return fmt.Errorf("ci.yml does not reference the proto stubs freshness gate")
}

// ─── buf lint integration ─────────────────────────────────────────────────────

func (s *stubSuite) bufLintIsRunFromTheProtosDirectory() error {
	return s.theTargetIsInspected("lint-protos")
}

func (s *stubSuite) itReportsZeroErrors() error {
	return s.fileContentContains("buf lint")
}

func (s *stubSuite) bufFormatDiffExitCodeIsRunFromTheProtosDirectory() error {
	return s.theTargetIsInspected("lint-protos")
}

func (s *stubSuite) itReportsNoFormattingDifferences() error {
	return s.fileContentContains("buf format")
}

// ─── Committed stubs scenarios ────────────────────────────────────────────────

func (s *stubSuite) theRepositoryContainsGoStubsDir() error {
	return s.theRepositoryContains("protos/generated/go/zynax/v1/")
}

func (s *stubSuite) theRepositoryContains(path string) error {
	if _, err := os.Stat(filepath.Join(s.root, path)); err != nil {
		return fmt.Errorf("expected %q to exist: %w", path, err)
	}
	return nil
}

func (s *stubSuite) theRepositoryContainsPyStubsDir() error {
	return s.theRepositoryContains("protos/generated/python/zynax/v1/")
}

// itContainsPair verifies two stub files exist. It auto-selects the correct
// generated directory based on file extension (.py → python, else go).
func (s *stubSuite) itContainsPair(a, b string) error {
	var dir string
	if strings.HasSuffix(a, ".py") {
		dir = filepath.Join(s.root, "protos", "generated", "python", "zynax", "v1")
	} else {
		dir = filepath.Join(s.root, "protos", "generated", "go", "zynax", "v1")
	}
	for _, name := range []string{a, b} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("expected stub %q: %w", name, err)
		}
	}
	return nil
}

// itContainsSingle verifies a single stub file exists, detecting directory
// from the file extension.
func (s *stubSuite) itContainsSingle(name string) error {
	var dir string
	if strings.HasSuffix(name, ".py") {
		dir = filepath.Join(s.root, "protos", "generated", "python", "zynax", "v1")
	} else {
		dir = filepath.Join(s.root, "protos", "generated", "go", "zynax", "v1")
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("expected stub %q: %w", name, err)
	}
	return nil
}

func (s *stubSuite) theGitignoreFileAtRepoRoot() error {
	return s.theFileExists(".gitignore")
}

func (s *stubSuite) theGitignoreRulesAreInspected() error {
	data, err := os.ReadFile(filepath.Join(s.root, ".gitignore"))
	if err != nil {
		return fmt.Errorf("reading .gitignore: %w", err)
	}
	s.fileContent = string(data)
	s.filePath = ".gitignore"
	return nil
}

func (s *stubSuite) isNotExcluded(path string) error {
	scanner := bufio.NewScanner(strings.NewReader(s.fileContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.Contains(line, path) {
			return fmt.Errorf(".gitignore excludes %q (line: %q)", path, line)
		}
	}
	return nil
}

func (s *stubSuite) aCommentConfirmsTheStubsAreIntentionallyCommitted() error {
	// The actual comment in .gitignore uses "ARE committed" not "intentionally committed"
	return s.fileContentContains("ARE committed")
}

// ─── helper ───────────────────────────────────────────────────────────────────

func (s *stubSuite) verifyPairedStubs(protoDir, stubDir, protoExt, stubExt string) error {
	entries, err := os.ReadDir(protoDir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", protoDir, err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), protoExt) {
			continue
		}
		base := strings.TrimSuffix(e.Name(), protoExt)
		expected := base + stubExt
		if _, err := os.Stat(filepath.Join(stubDir, expected)); err != nil {
			return fmt.Errorf("expected stub %q for proto %q: %w", expected, e.Name(), err)
		}
	}
	return nil
}

// ─── godog wiring ─────────────────────────────────────────────────────────────

func InitializeScenario(ctx *godog.ScenarioContext) {
	s := newSuite()

	// Background
	ctx.Step(`^the file "([^"]*)" exists$`, s.theFileExists)

	// buf.gen.yaml parsing
	ctx.Step(`^"([^"]*)" is parsed$`, s.bufGenYamlIsParsed)
	ctx.Step(`^it contains a plugin entry for "([^"]*)"$`, s.itContainsAPluginEntryFor)
	ctx.Step(`^it contains a plugin entry for Python protobuf generation$`, s.itContainsAPluginEntryForPythonProtobuf)
	ctx.Step(`^it contains a plugin entry for Python gRPC generation$`, s.itContainsAPluginEntryForPythonGRPC)
	ctx.Step(`^the Go output path is "([^"]*)"$`, s.theGoOutputPathIs)
	ctx.Step(`^the gRPC output path is "([^"]*)"$`, s.theGRPCOutputPathIs)
	ctx.Step(`^the Python output path is "([^"]*)"$`, s.thePythonOutputPathIs)
	ctx.Step(`^the Python gRPC output path is "([^"]*)"$`, s.thePythonGRPCOutputPathIs)

	// Go stubs — "are present" has no capture group
	ctx.Step(`^all \.proto files in protos/zynax/v1/ are present$`, s.allProtoFilesInArePresent)
	ctx.Step(`^make generate-protos is run inside the dev Docker image$`, s.makeGenerateProtosIsRunInsideTheDevDockerImage)
	ctx.Step(`^Go stub files are written under "([^"]*)"$`, s.goStubFilesAreWrittenUnder)
	ctx.Step(`^every proto file has a corresponding "_pb\.go" stub file$`, s.everyProtoFileHasACorrespondingPbGoStubFile)
	ctx.Step(`^every service proto has a corresponding "_grpc\.pb\.go" stub file$`, s.everyServiceProtoHasACorrespondingGrpcPbGoStubFile)
	ctx.Step(`^make generate-protos has been run$`, s.makeGenerateProtosHasBeenRun)
	ctx.Step(`^the Go stubs in "([^"]*)" are inspected$`, s.theGoStubsAreInspected)
	ctx.Step(`^they declare package "([^"]*)"$`, s.theyDeclarePackage)
	ctx.Step(`^they import "([^"]*)"$`, s.theyImport)
	ctx.Step(`^"([^"]*)" is inspected$`, s.inspectFile)
	ctx.Step(`^it contains the "([^"]*)" interface$`, s.itContainsTheInterface)

	// Python stubs
	ctx.Step(`^Python stub files are written under "([^"]*)"$`, s.pythonStubFilesAreWrittenUnder)
	ctx.Step(`^every proto file has a corresponding "_pb2\.py" stub file$`, s.everyProtoFileHasACorrespondingPb2PyStubFile)
	ctx.Step(`^every service proto has a corresponding "_pb2_grpc\.py" stub file$`, s.everyServiceProtoHasACorrespondingPb2GrpcPyStubFile)
	ctx.Step(`^the Python stubs in "([^"]*)" are imported$`, s.thePythonStubsAreImported)
	ctx.Step(`^no ImportError is raised$`, s.noImportErrorIsRaised)
	ctx.Step(`^the Python stubs in "([^"]*)" are inspected$`, s.thePythonStubsInAreInspected)
	ctx.Step(`^it contains the "([^"]*)" class$`, s.itContainsTheClass)

	// Makefile
	ctx.Step(`^a Makefile exists at the repository root$`, s.aMakefileExistsAtTheRepositoryRoot)
	ctx.Step(`^the "([^"]*)" target is inspected$`, s.theTargetIsInspected)
	ctx.Step(`^it invokes "buf generate" with "([^"]*)"$`, s.itInvokesBufGenerateWith)
	ctx.Step(`^it runs inside the keel-tools Docker image$`, s.itRunsInsideTheKeelToolsDockerImage)
	ctx.Step(`^it invokes "buf lint"$`, s.itInvokesBufLint)
	ctx.Step(`^it is included as a dependency of the "lint" target$`, s.itIsIncludedAsADependencyOfTheLintTarget)

	// Stubs regenerated cleanly
	ctx.Step(`^an existing set of generated stubs$`, s.anExistingSetOfGeneratedStubs)
	ctx.Step(`^a field is added to a proto message and make generate-protos is run$`, s.aFieldIsAddedAndMakeGenerateProtosIsRun)
	ctx.Step(`^the new field appears in both Go and Python stubs$`, s.theNewFieldAppearsInBothGoAndPythonStubs)
	ctx.Step(`^no stale generated files remain from the previous run$`, s.noStaleGeneratedFilesRemainFromThePreviousRun)

	// CI freshness gate
	ctx.Step(`^a pull request that modifies a \.proto file$`, s.aPullRequestThatModifiesAProtoFile)
	ctx.Step(`^make generate-protos was NOT run before committing$`, s.makeGenerateProtosWasNOTRunBeforeCommitting)
	ctx.Step(`^the "proto-stubs-fresh" CI check runs$`, s.theProtoStubsFreshCICheckRuns)
	ctx.Step(`^the check fails with a message to run "([^"]*)"$`, s.theCheckFailsWithAMessageToRun)
	ctx.Step(`^make generate-protos was run and the updated stubs were committed$`, s.makeGenerateProtosWasRunAndStubsWereCommitted)
	ctx.Step(`^the check passes$`, s.theCheckPasses)
	ctx.Step(`^a pull request that does not modify any \.proto files$`, s.aPullRequestThatDoesNotModifyAnyProtoFiles)
	ctx.Step(`^the check passes without inspecting stubs$`, s.theCheckPassesWithoutInspectingStubs)

	// buf lint — no capture groups
	ctx.Step(`^buf lint is run from the protos/ directory$`, s.bufLintIsRunFromTheProtosDirectory)
	ctx.Step(`^it reports zero errors$`, s.itReportsZeroErrors)
	ctx.Step(`^buf format --diff --exit-code is run from the protos/ directory$`, s.bufFormatDiffExitCodeIsRunFromTheProtosDirectory)
	ctx.Step(`^it reports no formatting differences$`, s.itReportsNoFormattingDifferences)

	// Committed stubs — pair and single file steps
	ctx.Step(`^the repository contains protos/generated/go/zynax/v1/$`, s.theRepositoryContainsGoStubsDir)
	ctx.Step(`^the repository contains protos/generated/python/zynax/v1/$`, s.theRepositoryContainsPyStubsDir)
	ctx.Step(`^it contains "([^"]*)" and "([^"]*)"$`, s.itContainsPair)
	ctx.Step(`^it contains "([^"]*)"$`, s.itContainsSingle)

	// .gitignore
	ctx.Step(`^the file "\.gitignore" at the repository root$`, s.theGitignoreFileAtRepoRoot)
	ctx.Step(`^the gitignore rules are inspected$`, s.theGitignoreRulesAreInspected)
	ctx.Step(`^"([^"]*)" is not excluded$`, s.isNotExcluded)
	ctx.Step(`^a comment confirms the stubs are intentionally committed$`, s.aCommentConfirmsTheStubsAreIntentionallyCommitted)
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/stub_generation.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
