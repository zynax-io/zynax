// SPDX-License-Identifier: Apache-2.0

package check

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// gencodeStubPath is a representative generated Python stub whose header records
// the protobuf gencode version ("# Protobuf Python Version: X.Y.Z"). Every stub
// is regenerated together, so one is authoritative for the whole tree.
const gencodeStubPath = "protos/generated/python/zynax/v1/agent_pb2.py"

var gencodeVersionRe = regexp.MustCompile(`(?m)^#\s*Protobuf Python Version:\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)

// ProtobufRuntimeViolation records a lockfile whose pinned protobuf runtime is
// older than the gencode version — the exact mismatch that crashes a Python
// adapter at import via runtime_version.ValidateProtobufRuntimeVersion.
type ProtobufRuntimeViolation struct {
	Lockfile string // repo-relative uv.lock path
	Runtime  string // pinned protobuf runtime version
	Gencode  string // gencode version the stubs require
}

// ProtobufRuntimeReport is the result of a ProtobufRuntime check.
type ProtobufRuntimeReport struct {
	Gencode    string
	Lockfiles  []string
	Violations []ProtobufRuntimeViolation
}

// ProtobufRuntime verifies that every Python uv.lock under agents/ pins a
// `protobuf` runtime >= the generated-code (gencode) version. A runtime older
// than gencode makes the generated stubs raise at import time via
// runtime_version.ValidateProtobufRuntimeVersion — the langgraph e2e crash the
// #1550 re-lock fixed. A lock that does not pin protobuf is skipped.
func ProtobufRuntime(repoRoot string) (ProtobufRuntimeReport, error) {
	gencode, err := readGencodeVersion(filepath.Join(repoRoot, gencodeStubPath))
	if err != nil {
		return ProtobufRuntimeReport{}, err
	}
	locks, err := findUVLocks(filepath.Join(repoRoot, "agents"))
	if err != nil {
		return ProtobufRuntimeReport{}, err
	}
	report := ProtobufRuntimeReport{Gencode: gencode}
	for _, lock := range locks {
		runtime, err := readLockedProtobuf(lock)
		if err != nil {
			return ProtobufRuntimeReport{}, err
		}
		report.Lockfiles = append(report.Lockfiles, relPath(repoRoot, lock))
		if runtime == "" {
			continue // lock does not pin protobuf — nothing to compare
		}
		if compareDottedVersions(runtime, gencode) < 0 {
			report.Violations = append(report.Violations, ProtobufRuntimeViolation{
				Lockfile: relPath(repoRoot, lock),
				Runtime:  runtime,
				Gencode:  gencode,
			})
		}
	}
	return report, nil
}

// PrintProtobufRuntimeReport writes a human-readable summary and returns true
// when no lockfile is below gencode.
func PrintProtobufRuntimeReport(w *os.File, r ProtobufRuntimeReport) bool {
	_, _ = fmt.Fprintf(w, "protobuf gencode version: %s (from %s)\n", r.Gencode, gencodeStubPath)
	_, _ = fmt.Fprintf(w, "checked %d Python lockfile(s)\n", len(r.Lockfiles))
	if len(r.Violations) == 0 {
		_, _ = fmt.Fprintln(w, "✅  All locked protobuf runtimes are >= the gencode version.")
		return true
	}
	for _, v := range r.Violations {
		_, _ = fmt.Fprintf(w, "❌  %s pins protobuf %s < gencode %s — the generated stubs raise at "+
			"import (ValidateProtobufRuntimeVersion). Re-lock protobuf to >= %s.\n",
			v.Lockfile, v.Runtime, v.Gencode, v.Gencode)
	}
	return false
}

// readGencodeVersion extracts the gencode version from a generated stub header.
func readGencodeVersion(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path derived from a repo-relative constant
	if err != nil {
		return "", fmt.Errorf("read gencode stub %s: %w", path, err)
	}
	m := gencodeVersionRe.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("no '# Protobuf Python Version:' header in %s", path)
	}
	return string(m[1]), nil
}

// findUVLocks walks base and returns every uv.lock path, skipping virtualenvs
// and hidden/cache dirs.
func findUVLocks(base string) ([]string, error) {
	var locks []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".venv" || name == "node_modules" || (strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "uv.lock" {
			locks = append(locks, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", base, err)
	}
	sort.Strings(locks)
	return locks, nil
}

// readLockedProtobuf returns the version pinned for the `protobuf` package in a
// uv.lock, or "" if the lock does not pin it. uv.lock is TOML; each package is a
// [[package]] table with adjacent `name` and `version` keys.
func readLockedProtobuf(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // path discovered by walking the repo tree
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	inProtobuf := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "[[package]]":
			inProtobuf = false
		case strings.HasPrefix(line, "name = "):
			inProtobuf = tomlStringValue(line) == "protobuf"
		case inProtobuf && strings.HasPrefix(line, "version = "):
			return tomlStringValue(line), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return "", nil
}

// tomlStringValue extracts the quoted value from a `key = "value"` TOML line.
func tomlStringValue(line string) string {
	i := strings.IndexByte(line, '"')
	if i < 0 {
		return ""
	}
	rest := line[i+1:]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// compareDottedVersions compares dotted numeric versions ("7.35.1"). It returns
// -1 if a < b, 0 if equal, +1 if a > b. Non-numeric segments compare as 0.
func compareDottedVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			ai, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(bs[i])
		}
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}
